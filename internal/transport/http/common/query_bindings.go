package common

type CleanType string

const (
	PleaseClean  CleanType = "PLEASE_CLEAN_THIS_ROOM"
	DoNotDisturb CleanType = "DO_NOT_DISTURB"
)

type NeedCleanQuery struct {
	Clean CleanType `query:"clean" binding:"required,oneof=PLEASE_CLEAN_THIS_ROOM DO_NOT_DISTURB"`
}
