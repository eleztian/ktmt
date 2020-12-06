package ktmt

import (
	"errors"
	"net"
	"strings"
)

var (
	ErrCanceled = errors.New("canceled")
	ErrClosed   = errors.New("isClosed")
)

func IsNetClosedErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}
func IsNetTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}
