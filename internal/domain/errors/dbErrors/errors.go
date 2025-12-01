package dbErrors

import (
	"errors"
	"fmt"
)

type CottageNameDoesNotExist struct {
	CottageName string
}

func (e *CottageNameDoesNotExist) Error() string {
	return fmt.Sprintf("Cottage with name %s does not exist", e.CottageName)
}

// ErrBookingRepo indicates an unexpected internal failure in the Booking repository.
var ErrBookingRepo = errors.New("bookingRepository failure")

// ErrCottageRepo indicates an unexpected internal failure in the Cottage repository.
var ErrCottageRepo = errors.New("cottageRepository failure")
