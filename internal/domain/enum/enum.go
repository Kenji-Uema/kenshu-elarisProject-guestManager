package enum

type CleaningStatus string
type KeyHolder string

const (
	FullyCleaned CleaningStatus = "fully_cleaned"
)

const (
	KeyHolderGuest   KeyHolder = "guest"
	KeyHolderCottage KeyHolder = "cottage"
)

type BookingStatus string

const (
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusPast      BookingStatus = "past"
)

type CleaningRequestType string

const (
	PrepareForGuest = CleaningRequestType("prepareForGuest")
	DailyCleaning   = CleaningRequestType("dailyCleaning")
	FullCleaning    = CleaningRequestType("fullCleaning")
	PrepareForSleep = CleaningRequestType("prepareForSleep")
)
