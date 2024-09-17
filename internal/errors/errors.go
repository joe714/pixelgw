package errors

import (
	ne "errors"
	"fmt"
)

type CodedError interface {
	Code() int32
}

type cerror struct {
	code int32
	msg  string
}

func New(code int32, text string) error {
	return &cerror{
		code: code,
		msg:  text,
	}
}

func (e *cerror) Error() string {
	return e.msg
}

func (e *cerror) Code() int32 {
	return e.code
}

type werror struct {
	base error
	msg  string
}

func Wrap(base error, format string, a ...any) error {
	return &werror{
		base: base,
		msg:  fmt.Sprintf(format, a...),
	}
}

func (e *werror) Error() string {
	return e.msg
}

func (e *werror) Unwrap() error {
	return e.base
}

func Code(e error) int32 {
	var c CodedError
	if ne.As(e, &c) {
		return c.Code()
	}
	return 1000
}

var (
	ChannelExists      = New(1001, "channel exists")
	ChannelNotFound    = New(1002, "channel not found")
	AppIndexOutOfRange = New(1011, "index out of range")
)
