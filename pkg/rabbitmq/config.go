package rabbitmq

type Config struct {
	URL          string
	Exchange     string
	ExchangeType string
	Queue        string
	RoutingKey   string
	ConsumerTag  string
}
