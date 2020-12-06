package client

import (
	"github.com/eleztian/ktmt"
	"net"
	"time"
)

type Options struct {
	id          string
	version     byte
	readTimeout time.Duration
	writeTimout time.Duration
	keepalive   uint16
	dialer      func() (net.Conn, error)
	onClose     ktmt.OnCloseCallback
}

type Dialer func() (net.Conn, error)

func NewOptions(id string, dialer Dialer, ops ...OptionFunc) *Options {
	res := &Options{
		id:     id,
		dialer: dialer,
	}

	for _, op := range ops {
		op(res)
	}
	return res
}

type OptionFunc func(ops *Options)

func WithReadTimeout(to time.Duration) OptionFunc {
	return func(ops *Options) {
		ops.readTimeout = to
	}
}

func WithWriteTimeout(to time.Duration) OptionFunc {
	return func(ops *Options) {
		ops.writeTimout = to
	}
}

func WithOnClose(f ktmt.OnCloseCallback) OptionFunc {
	return func(ops *Options) {
		ops.onClose = f
	}
}

func WithKeepalive(seconds uint16) OptionFunc {
	return func(ops *Options) {
		ops.keepalive = seconds
	}
}

func WithVersion(version byte) OptionFunc {
	return func(ops *Options) {
		ops.version = version
	}
}
