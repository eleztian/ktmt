package server

import (
	"context"
	"git.moresec.cn/zhangtian/ktmt"
	"git.moresec.cn/zhangtian/ktmt/mock"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewApp_Close_Open(t *testing.T) {
	c := mock.NewConn()
	sc := c.Server
	cc := c.Client
	ln := mock.NewListener(sc, cc)
	app, err := NewApp(
		ktmt.NewJsonPresentationLayer(),
		NewOptions(ln),
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

	ln = mock.NewListener(sc, cc)
	app, err = NewApp(
		ktmt.NewJsonPresentationLayer(),
		NewOptions(ln),
	)
	if err != nil {
		t.Error(err)
		return
	}
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
	err = app.Close()
	if err != nil {
		t.Error(err)
		return
	}
}

func mockClient(conn net.Conn) error {
	_, err := ktmt.Connect(conn, &ktmt.ConnConfig{
		Version:      0,
		ReadTimeout:  0,
		WriteTimeout: 0,
		CID:          "test",
		Keepalive:    10,
		CleanSession: false,
		CloseCB:      nil,
	})
	if err != nil {
		return err
	}

	return nil
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

	return nil
}

func TestApp_AcceptSession(t *testing.T) {
	c := mock.NewConn()
	sc := c.Server
	cc := c.Client
	ln := mock.NewListener(sc, cc)
	app, err := NewApp(
		ktmt.NewJsonPresentationLayer(),
		NewOptions(ln),
	)
	if err != nil {
		t.Error(err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = app.Open(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := mockClient(cc)
		if err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
	time.Sleep(time.Microsecond * 500)
	if len(app.Sessions()) != 1 {
		t.Error("session reg")
		return
	}

	err = app.Close()
	if err != nil {
		t.Error(err)
		return
	}
	if len(app.Sessions()) != 0 {
		t.Error("session reg")
		return
	}
}
