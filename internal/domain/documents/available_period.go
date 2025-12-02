package documents

import (
	"time"
)

type CottageAvailablePeriod struct {
	Name    string
	Periods []Period
}

type Period struct {
	Start time.Time
	End   time.Time
}
