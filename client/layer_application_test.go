package client

import (
	"context"
	"errors"
	"github.com/eleztian/ktmt"
	"github.com/eleztian/ktmt/mock"
	"github.com/eleztian/ktmt/packets"
	"net"
	"sync"
	"testing"
)

func TestNewApp_Close_Open(t *testing.T) {
	c := mock.NewConn()
	sc := c.Server
	cc := c.Client
	dialer := func() (net.Conn, error) {
		return cc, nil
	}
	app, err := NewApp(
		ktmt.NewJsonPresentationLayer(),
		NewOptions("test-id", dialer),
	)
	if err != nil {
		t.Error(err)
		return
	}

	err = app.Close()
	if err != nil {
		t.Error(err)
		return
	}
	app, err = NewApp(
		ktmt.NewJsonPresentationLayer(),
		NewOptions("test-id", dialer),
	)
	if err != nil {
		t.Error(err)
		return
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = mockServer(sc)
		if err != nil {
			t.Error(err)
			return
		}
	}()
	err = app.Open(context.TODO())
	if err != nil {
		t.Error(err)
		return
	}
	err = app.Open(context.TODO())
	if err == nil || err.Error() != "is opened" {
		t.Error("open error")
		return
	}

	err = app.Close()
	if err != nil {
		t.Error(err)
		return
	}

	wg.Wait()
	err = app.Close()
	if err != nil {
		t.Error(err)
		return
	}
}

func mockServer(conn net.Conn) error {
	_, err := ktmt.Accept(conn, &ktmt.ConnConfig{
		Version:      0,
		ReadTimeout:  0,
		WriteTimeout: 0,
		CID:          "",
		Keepalive:    0,
		CleanSession: false,
		CloseCB:      nil,
	})
	if err != nil {
		return err
	}

	cp, err := packets.ReadPacket(conn)
	if err != nil {
		return err
	}

	if _, ok := cp.(*packets.DisconnectPacket); !ok {
		return errors.New("not a disconnect pkt")
	}

	return nil
}
