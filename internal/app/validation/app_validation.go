package validation

import (
	"regexp"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
)

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[A-Za-z]{2,}$`)

func (v *Validator) EmailFormat(email string) *Validator {
	v.steps = append(v.steps, func() error {
		if !emailRe.MatchString(email) {
			return &validationErrors.ErrValidationConstrain{
				Field:   "email",
				Message: "email not valid",
			}
		}
		return nil
	})

	return v
}

func (v *Validator) Period(start time.Time, end time.Time) *Validator {
	v.steps = append(v.steps, func() error {
		if start.After(end) || start.Equal(end) {
			return &validationErrors.ErrValidationConstrain{
				Field: "start", Message: "start date must be before end date"}
		}
		return nil
	})
	return v
}
