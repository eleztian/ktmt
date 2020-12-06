package server

import (
	"net"
	"time"
)

type Options struct {
	ReadTimeout time.Duration
	WriteTimout time.Duration

	ln net.Listener

	OnOnlineFunc  func(cid string) error
	OnOfflineFunc func(cid string) error
}

func NewOptions(ln net.Listener, ops ...OptionFunc) *Options {
	res := &Options{
		ln:            ln,
		ReadTimeout:   10 * time.Second,
		WriteTimout:   10 * time.Second,
		OnOnlineFunc:  defaultOnOnline,
		OnOfflineFunc: defaultOnOffline,
	}

	for _, op := range ops {
		op(res)
	}

	return res
}

type OptionFunc func(ops *Options)

func WithReadTimeout(to time.Duration) OptionFunc {
	return func(ops *Options) {
		ops.ReadTimeout = to
	}
}

func WithWriteTimeout(to time.Duration) OptionFunc {
	return func(ops *Options) {
		ops.WriteTimout = to
	}
}

func WithOnOnline(f func(string) error) OptionFunc {
	return func(ops *Options) {
		ops.OnOnlineFunc = f
	}
}

func WithOnOffline(f func(string) error) OptionFunc {
	return func(ops *Options) {
		ops.OnOfflineFunc = f
	}
}

func defaultOnOnline(cid string) error {
	return nil
}

func defaultOnOffline(cid string) error {
	return nil
}
