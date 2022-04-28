package rabbitmq

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

const delay = 3

type ChannelAmqp struct {
	*amqp.Channel
	closed     int32
	process    int32
	stopSignal int32
}

type PublishChannel interface {
	Publish(exchange string, routingKey string, body []byte) error
}

// Connection -
type Connection struct {
	conn    *amqp.Connection
	channel *ChannelAmqp
	config  Config
	Finish  chan bool
}

// Processor -
type Processor interface {
	Process(routingKey string, bytes []byte) error
	Fallback(bytes []byte, err error)
}

const (
	connectionClose     = "Connection closed. Reconnecting..."
	connectionCloseInfo = "Connection closed info"
	connectionSuccess   = "Reconnecting success."
	connectionFail      = "Reconnecting failed."

	channelClose     = "Channel closed. Re-running init..."
	channelCloseInfo = "Channel closed info"
	channelSuccess   = "Recreate channel success."
	channelFail      = "Recreate channel failed."
)

// NewConnection wrap `amqp.Dial`, enhance feature re-connection
func NewConnection(config Config) (*Connection, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, err
	}

	connection := &Connection{
		conn:   conn,
		config: config,
		Finish: config.Finish,
	}

	connection.startConnection()
	ch, err := connection.conn.Channel()
	if err != nil {
		return nil, err
	}

	ch.Qos(config.PrefetchCount, config.PrefetchSize, config.Global)
	channel := &ChannelAmqp{
		Channel:    ch,
		process:    0,
		stopSignal: 0,
	}
	connection.channel = channel
	connection.startChannel()

	return connection, nil
}

func (c *Connection) startConnection() {
	go func() {
		for {
			reason, ok := <-c.conn.NotifyClose(make(chan *amqp.Error))
			if !ok {
				logger.Info("", connectionClose)
				break
			}
			logger.Info("", reason.Error())

			// reconnect
			for {
				time.Sleep(delay * time.Second)
				conn, err := amqp.Dial(c.config.URL)
				if err == nil {
					c.conn = conn
					logger.Info("", connectionSuccess)
					break
				}
				logger.Error("", connectionFail+err.Error(), err)
			}
		}
	}()
}

func (c *Connection) startChannel() {
	go func() {
		for {
			reason, ok := <-c.channel.Channel.NotifyClose(make(chan *amqp.Error))
			if !ok || c.channel.IsClosed() {
				logger.Info("", channelClose)
				err := c.channel.Close() // close again, ensure closed flag set when connection closed
				if err != nil {
					logger.Info("", channelFail)
				}

				break
			}
			logger.Info("", reason.Error())

			// reconnect
			for {
				time.Sleep(delay * time.Second)

				ch, err := c.conn.Channel()
				if err == nil {
					logger.Info("", channelSuccess)
					c.channel.Channel = ch
					err = ch.Qos(c.config.PrefetchCount, c.config.PrefetchSize, c.config.Global)
					if err != nil {
						logger.Warn(err)
						continue
					}

					break
				}

				logger.Error("", channelFail+err.Error(), err)
			}
		}
	}()
}

// Consume -
func (c *Connection) Consume(processor Processor) {
	go func() {
		for {
			tasks, err := c.channel.Channel.Consume(c.config.Queue, c.config.Queue, false, false, false, false, nil)
			if err != nil {
				logger.Error("consume fail", err)

				if c.channel.ShouldStop() {
					break
				}

				time.Sleep(time.Second * 5)
				continue
			}

			for task := range tasks {
				c.channel.SetProcess(1)
				err := processor.Process(task.RoutingKey, task.Body)
				if err != nil {
					err = c.Retry(task)
					if err != nil {
						processor.Fallback(task.Body, err)
					}
				}
				task.Ack(true)
				c.channel.SetProcess(0)
				if c.channel.ShouldStop() {
					c.CloseConnection()
				}
			}
		}
	}()
}

func (c *Connection) Retry(task amqp.Delivery) error {
	retryCount := 0
	if task.Headers["x-retries"] != nil {
		retryCount = int(task.Headers["x-retries"].(int32))
	}
	retryTime := time.Second * time.Duration(c.config.MaxRetrySeconds)
	expireTime := time.Now().Add(retryTime).Unix()

	if task.Headers["x-expire"] != nil {
		expireTime = task.Headers["x-expire"].(int64)
	}

	retryDelay := c.config.RetryDelay * int(math.Pow(2, float64(retryCount)))

	if task.Headers["x-delay"] != nil && task.Headers["x-retries"] != nil {
		retryDelay = c.config.RetryDelay * int(math.Pow(2, float64(task.Headers["x-retries"].(int32))))
	}

	if expireTime > time.Now().Unix()+int64(retryDelay) {
		err := c.channel.Channel.Publish(c.config.RetryExchange, task.RoutingKey, true, false, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Headers: map[string]interface{}{
				"x-retries": retryCount + 1,
				"x-expire":  expireTime,
				"x-delay":   retryDelay,
			},
			Timestamp: task.Timestamp,
			Body:      task.Body,
		})
		if err != nil {
			logger.Info("Error can't publish queue")
			return err
		}
	} else {
		logger.Info("max retries reached.")
		return fmt.Errorf("max_retries_reached")
	}
	return nil
}

func (c *Connection) CancelConsume() {
	c.channel.Cancel(c.config.Queue, false)
	c.channel.SetShouldStop(1)

	if !c.channel.IsProcessing() {
		c.CloseConnection()
	}
}

func (c *Connection) CloseConnection() {
	c.channel.Close()
	c.conn.Close()
	c.Finish <- true
}

func (ch *ChannelAmqp) ShouldStop() bool {
	return atomic.LoadInt32(&ch.stopSignal) == 1
}

func (ch *ChannelAmqp) SetShouldStop(i int32) {
	atomic.StoreInt32(&ch.stopSignal, i)
}

func (ch *ChannelAmqp) IsProcessing() bool {
	return atomic.LoadInt32(&ch.process) == 1
}

func (ch *ChannelAmqp) SetProcess(i int32) {
	atomic.StoreInt32(&ch.process, i)
}

func (ch *ChannelAmqp) IsClosed() bool {
	return atomic.LoadInt32(&ch.closed) == 1
}

// Close ensure closed flag set
func (ch *ChannelAmqp) Close() error {
	if ch.IsClosed() {
		return amqp.ErrClosed
	}

	atomic.StoreInt32(&ch.closed, 1)
	return ch.Channel.Close()
}

func (ch *ChannelAmqp) Publish(exchange string, routingKey string, body []byte) error {
	err := ch.Channel.Publish(exchange, routingKey, true, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	return err
}

func (c *Connection) GetChannel() PublishChannel {
	return c.channel
}
