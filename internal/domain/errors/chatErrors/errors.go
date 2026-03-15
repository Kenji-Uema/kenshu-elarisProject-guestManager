package chatErrors

import "fmt"

type AckNotReceivedErr struct {
	Err error
}

func (e *AckNotReceivedErr) Error() string {
	return fmt.Sprintf("ack not received: %v", e.Err)
}

type ReplyNotReceivedErr struct {
	Err error
}

func (e *ReplyNotReceivedErr) Error() string {
	return fmt.Sprintf("reply not received: %v", e.Err)
}
