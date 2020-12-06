package client

import (
	"context"
	"errors"
	"github.com/eleztian/ktmt"
	"sync"
	"time"
)

type client struct {
	sync.RWMutex
	opened bool
	ktmt.Router
	pl      ktmt.PresentationLayer
	session ktmt.SessionLayer

	options *Options
}

func NewApp(pl ktmt.PresentationLayer, op *Options) (ktmt.ApplicationLayer, error) {

	return &client{
		RWMutex: sync.RWMutex{},
		Router:  ktmt.NewRouter(),
		pl:      pl,
		options: op,
	}, nil

}

func (c *client) setOpen() {
	c.Lock()
	c.opened = true
	c.Unlock()
}

func (c *client) isOpened() bool {
	c.RLock()
	res := c.opened
	c.RUnlock()
	return res
}

func (c *client) setSession(sl ktmt.SessionLayer) {
	c.Lock()
	c.session = sl
	c.Unlock()
}

func (c *client) Open(ctx context.Context) error {
	if c.isOpened() {
		return errors.New("is opened")
	}
	cl := ktmt.NewConnectionKeeper(ctx, func(ctx context.Context) ktmt.MConn {
		var res ktmt.MConn
		var count = 1
		for {
			conn, err := c.options.dialer()
			if err != nil {
				if count < 10 {
					count++
				}
				time.Sleep(time.Duration(count) * time.Second)
				continue
			}
			count = 2
			res, err = ktmt.Connect(conn, &ktmt.ConnConfig{
				Version:      c.options.version,
				ReadTimeout:  c.options.readTimeout,
				WriteTimeout: c.options.writeTimout,
				CID:          c.options.id,
				Keepalive:    c.options.keepalive,
				CleanSession: false,
				CloseCB:      c.options.onClose,
			})
			if err != nil {
				_ = conn.Close()
				time.Sleep(time.Duration(count) * time.Second)
				continue
			}
			break
		}
		return res
	}, true)

	sl, err := ktmt.NewSession(ctx, cl)
	if err != nil {
		return err
	}

	c.setOpen()
	c.setSession(sl)

	inMessages := make(chan ktmt.Message)
	go func() {
		defer close(inMessages)
		for {
			select {
			case m, ok := <-c.session.In():
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case inMessages <- m:
				}

			case <-ctx.Done():

			}
		}

	}()

	go c.Router.MatchAndDispatch(inMessages, false)

	return nil
}

func (c *client) Publish(ctx context.Context, topic string, data interface{}) error {
	content, err := c.pl.Encode(data)
	if err != nil {
		return err
	}

	tk := c.session.Send(ctx, topic, 1, content)
	tk.Wait()

	return tk.Error()
}

func (c *client) PublishTimeout(ctx context.Context, topic string, data interface{}, timeout time.Duration) error {
	content, err := c.pl.Encode(data)
	if err != nil {
		return err
	}

	tk := c.session.Send(ctx, topic, 1, content)
	tk.WaitTimeout(timeout)

	return tk.Error()
}

func (c *client) PublishWithID(ctx context.Context, _ string, topic string, data interface{}) (tk ktmt.Token) {
	content, err := c.pl.Encode(data)
	if err != nil {
		tk = ktmt.NewErrToken(err)
		return
	}
	tk = c.session.Send(ctx, topic, 1, content)
	return tk
}

func (c *client) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.session != nil {
		c.session.Close()
	}

	return nil
}

func (c *client) Sessions() []string {
	return []string{}
}

func (c *client) Remove(string) {

}
