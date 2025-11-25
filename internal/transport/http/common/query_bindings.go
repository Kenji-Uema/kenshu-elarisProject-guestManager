package common

type NeedCleanQuery struct {
	Clean string `query:"clean" binding:"required,oneof=PLEASE_CLEAN_THIS_ROOM DO_NOT_DISTURB"`
}
