package domain

type RabbitMqMessage struct {
	Exchange    string
	Key         string
	ContentType string
	Body        []byte
}
