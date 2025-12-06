package appErrors

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ErrGuestNotFound struct {
	Id         primitive.ObjectID
	DocumentId string
}

func (e *ErrGuestNotFound) Error() string {
	if strings.TrimSpace(e.DocumentId) != "" {
		return fmt.Sprintf("Guest with document id %s does not exist", e.DocumentId)
	}
	return fmt.Sprintf("Guest with id %s does not exist", e.Id.Hex())
}
