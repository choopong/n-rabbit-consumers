package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"choopong.com/n-rabbit-consumers/cmd/worker/processor"
	"choopong.com/n-rabbit-consumers/pkg/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	logger "github.com/sirupsen/logrus"
)

// Load .env if in dev environment
func init() {
	if currentEnvironment, ok := os.LookupEnv("ENV"); ok {
		if currentEnvironment == "dev" {
			err := godotenv.Load("./.env")
			if err != nil {
				logger.Info("Can't load .env", err)
			}
		}
	}
}

var rabbitMQURI string
var rabbitMQName string
var rabbitMQRetryExchange string
var numWorkers int

func init() {
	rabbitMQURI = os.Getenv("RABBIT_MQ_URI")
	rabbitMQName = os.Getenv("RABBIT_MQ_WORKER_QUEUE_NAME")
	rabbitMQRetryExchange = os.Getenv("RABBIT_MQ_MANAGER_RETRY_EXCHANGE")
	numWorkers, _ = strconv.Atoi(os.Getenv("NUM_WORKERS"))
}

func main() {
	qConn, err := rabbitmq.NewConnection(rabbitmq.Config{
		URL:               rabbitMQURI,
		Queue:             rabbitMQName,
		Delay:             3,
		RetryExchange:     rabbitMQRetryExchange,
		MaxRetrySeconds:   300,
		RetryDelay:        100,
		Finish:            make(chan bool),
		PrefetchCount:     1,
		PrefetchSize:      0,
		Global:            true,
		MultipleConsumers: true,
		NumConsumers:      numWorkers,
	})
	if err != nil {
		logger.Error("rabbitmq.NewConnection Error", err)
	}

	cProcessor := processor.NewProcessor()
	qConn.Consume(cProcessor)

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Run(":3000")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-exit
}
