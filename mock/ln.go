package mock

import (
	"errors"
	"io"
	"net"
	"time"
)

type listener struct {
	close  bool
	get    bool
	server net.Conn
	client net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	if l.close {
		return nil, errors.New("closed")
	}
	if !l.get {
		l.get = true
		return l.server, nil
	}
	for !l.close {
		time.Sleep(time.Second)
	}
	return nil, io.EOF
}

func (l *listener) Close() error {
	l.close = true
	return nil
}

func (l *listener) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
		Zone: "",
	}
}

func NewListener(server, client net.Conn) net.Listener {
	return &listener{
		close:  false,
		get:    false,
		server: server,
		client: client,
	}
}
