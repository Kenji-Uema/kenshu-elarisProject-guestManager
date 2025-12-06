package domain

import (
	"guestManager/internal/app/validation"
	"time"
)

type CottageAvailablePeriod struct {
	name    string
	periods []Period
}

type Period struct {
	start time.Time
	end   time.Time
}

func NewPeriod(start, end time.Time) (Period, error) {
	if err := validation.New().Period(start, end).Validate(); err != nil {
		return Period{}, err
	}

	return Period{
		startOfDay(start),
		startOfDay(end.AddDate(0, 0, 1)).Add(-time.Nanosecond)}, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
