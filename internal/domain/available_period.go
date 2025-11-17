package domain

import (
	"time"
)

type Period struct {
	Start time.Time
	End   time.Time
}

type CottageAvailablePeriod struct {
	Name    string
	Periods []Period
}

func (p *Period) Valid() bool {
	return p.Start.Before(p.End)
}

func (p *Period) Normalize() {
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	p.Start = startOfDay(p.Start)
	p.End = startOfDay(p.End.AddDate(0, 0, 1)).Add(-time.Nanosecond)
}
