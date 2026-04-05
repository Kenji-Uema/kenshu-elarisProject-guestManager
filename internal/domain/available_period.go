package domain

import (
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
)

type Period struct {
	CheckIn  time.Time
	CheckOut time.Time
}

func NewPeriod(start, end time.Time) (Period, error) {
	if err := validation.New().Period(start, end).Validate(); err != nil {
		return Period{}, err
	}

	return Period{
		startOfDay(start),
		startOfDay(end),
	}, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
