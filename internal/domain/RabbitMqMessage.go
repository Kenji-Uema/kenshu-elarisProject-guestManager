package domain

import "guestManager/internal/app/validation"

type RabbitMqMessage struct {
	exchange    string
	key         string
	contentType string
	body        []byte
}

func NewRabbitMqMessage(exchange string, key string, contentType string, body []byte) (RabbitMqMessage, error) {
	if err := validation.New().ExchangeName(exchange).AllowedContentType(contentType).Validate(); err != nil {
		return RabbitMqMessage{}, err
	}

	return RabbitMqMessage{exchange, key, contentType, body}, nil
}

func (m RabbitMqMessage) Exchange() string {
	return m.exchange
}

func (m RabbitMqMessage) Key() string {
	return m.key
}

func (m RabbitMqMessage) ContentType() string {
	return m.contentType
}

func (m RabbitMqMessage) Body() []byte {
	return m.body
}
