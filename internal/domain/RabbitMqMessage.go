package domain

import (
	"fmt"
	"guestManager/internal/domain/errors/appErrors"
	"regexp"
	"slices"
)

type RabbitMqMessage struct {
	exchange    string
	key         string
	contentType string
	body        []byte
}

var exchangeNameRe = regexp.MustCompile(`^ex\.[A-Za-z0-9_-]+$`)
var allowedContentTypes = []string{"application/json"}

func NewRabbitMqMessage(exchange string, key string, contentType string, body []byte) (RabbitMqMessage, error) {
	if exchangeNameRe.MatchString(exchange) {
		return RabbitMqMessage{}, &appErrors.ErrValidationConstrain{
			Field:   "exchange",
			Message: "must be in format ex.<name of exchange>",
		}
	}

	if !slices.Contains(allowedContentTypes, contentType) {
		return RabbitMqMessage{}, &appErrors.ErrValidationConstrain{
			Field:   "contentType",
			Message: fmt.Sprintf("must be of those: %q", allowedContentTypes),
		}
	}

	return RabbitMqMessage{exchange, key, contentType, body}, nil
}
