package validation

import (
	"guestManager/internal/domain/errors/validationErrors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (v *Validator) NotNilObjectID(field string, id primitive.ObjectID) *Validator {
	v.steps = append(v.steps, func() error {
		if id == primitive.NilObjectID {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be nil"}
		}
		return nil
	})
	return v
}
