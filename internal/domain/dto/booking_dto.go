package dto

import "time"

type BookingDto struct {
	NumberOfGuests int    `json:"number_of_guests"`
	StayPeriod     Period `json:"stay_period"`
	CottageName    string `json:"cottage_name"`
	Status         string `json:"status"`
}

type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
