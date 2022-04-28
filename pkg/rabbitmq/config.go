package rabbitmq

type Config struct {
	Queue             string
	RetryExchange     string
	MaxRetrySeconds   int
	RetryDelay        int
	Delay             int
	URL               string
	Finish            chan bool
	PrefetchCount     int
	PrefetchSize      int
	Global            bool
	MultipleConsumers bool
	NumConsumers      int
}
