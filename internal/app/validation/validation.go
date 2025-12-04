package validation

import (
	"guestManager/internal/domain/errors/appErrors"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Validator struct {
	steps []func() error
}

func New() *Validator {
	return &Validator{
		steps: make([]func() error, 0),
	}
}

func (v *Validator) NotBlank(field, value string) *Validator {
	v.steps = append(v.steps, func() error {
		if strings.TrimSpace(value) == "" {
			return &appErrors.ErrValidationConstrain{Field: field, Message: "must not be blank"}
		}
		return nil
	})
	return v
}

func (v *Validator) NotZeroValue(field string, value any) *Validator {
	v.steps = append(v.steps, func() error {
		if reflect.ValueOf(value).IsZero() {
			return &appErrors.ErrValidationConstrain{Field: field, Message: "must not be zero value"}
		}
		return nil
	})
	return v
}

func (v *Validator) NotNilObjectID(field string, id primitive.ObjectID) *Validator {
	v.steps = append(v.steps, func() error {
		if id == primitive.NilObjectID {
			return &appErrors.ErrValidationConstrain{Field: field, Message: "must not be nil"}
		}
		return nil
	})
	return v
}

func (v *Validator) Validate() error {
	for _, step := range v.steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
