package server

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/eleztian/ktmt"
)

type server struct {
	ktmt.Router
	sync.RWMutex
	opened bool

	sessions sync.Map

	ln net.Listener
	pl ktmt.PresentationLayer

	closed  chan struct{}
	options *Options
}

func (s *server) setOpen() {
	s.Lock()
	s.opened = true
	s.Unlock()
}

func (s *server) isOpened() bool {
	s.RLock()
	res := s.opened
	s.RUnlock()
	return res
}

func (s *server) Open(ctx context.Context) error {
	if s.isOpened() {
		return errors.New("is opened")
	}
	s.setOpen()

	inMessages := make(chan ktmt.Message)
	go func() {
		ctxTask, cancel := context.WithCancel(ctx)
		defer cancel()

		for {
			conn, err := s.ln.Accept()
			if err != nil {
				if !ktmt.IsNetClosedErr(err) {
					ktmt.ERROR.Printf("[APP] Accept %v", err)
				}
				break
			}

			go func() {
				defer conn.Close()
				session, err := s.newSession(ctxTask, conn)
				if err != nil {
					ktmt.ERROR.Printf("[APP] New Session %v", err)
					return
				}
				for pkt := range session.In() {
					select {
					case inMessages <- pkt:
					case <-ctx.Done():
					}
				}
			}()
		}

		s.sessions.Range(func(key, value interface{}) bool {
			s.delSession(key.(string))
			return true
		})

		close(inMessages)
		close(s.closed)
	}()

	go s.Router.MatchAndDispatch(inMessages, false)

	return nil
}

func (s *server) Close() error {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	_ = s.ln.Close()

	if s.opened {
		<-s.closed
	}

	return nil
}

func (s *server) Sessions() []string {
	res := make([]string, 0)
	s.sessions.Range(func(key, value interface{}) bool {
		res = append(res, key.(string))
		return true
	})
	return res
}

func (s *server) Remove(sid string) {
	s.delSession(sid)
}

func (s *server) Publish(ctx context.Context, topic string, data interface{}) error {
	content, err := s.pl.Encode(data)
	if err != nil {
		return err
	}

	sids := s.Sessions()

	for _, sid := range sids {
		sl := s.getSession(sid)
		if sl == nil {
			return errors.New("not found client")
		}

		tk := sl.Send(ctx, topic, 1, content)
		tk.Wait()
		if err := tk.Error(); err != nil {
			return err
		}
	}

	return nil
}

func (s *server) PublishTimeout(ctx context.Context, topic string, data interface{}, timeout time.Duration) error {
	content, err := s.pl.Encode(data)
	if err != nil {
		return err
	}

	sids := s.Sessions()

	for _, sid := range sids {
		sl := s.getSession(sid)
		if sl == nil {
			return errors.New("not found client")
		}

		tk := sl.Send(ctx, topic, 1, content)
		tk.WaitTimeout(timeout)
		if err := tk.Error(); err != nil {
			return err
		}
	}

	return nil
}

func (s *server) PublishWithID(ctx context.Context, sid string, topic string, data interface{}) (tk ktmt.Token) {
	content, err := s.pl.Encode(data)
	if err != nil {
		tk = ktmt.NewErrToken(err)
		return
	}
	sl := s.getSession(sid)
	if sl == nil {
		tk = ktmt.NewErrToken(errors.New("not found client"))
		return
	}

	tk = sl.Send(ctx, topic, 1, content)

	return
}

func (s *server) newSession(ctx context.Context, conn net.Conn) (ktmt.SessionLayer, error) {
	cl := ktmt.NewConnectionKeeper(ctx, func(ctx context.Context) ktmt.MConn {
		mc, err := ktmt.Accept(conn, &ktmt.ConnConfig{
			ReadTimeout:  s.options.ReadTimeout,
			WriteTimeout: s.options.WriteTimout,
			CloseCB: func(id string) {
				s.delSession(id)
			},
		})

		if err != nil {
			_ = conn.Close()
			if !ktmt.IsNetClosedErr(err) {
				ktmt.ERROR.Printf("[APP] Accept %s %v", conn.RemoteAddr().String(), err)
			}

			return nil
		}
		if mc.CleanSession() {
			s.sessions.Delete(mc.ID())
		}
		return mc
	}, false)

	sl, err := s.addSession(ctx, cl.ID(), cl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

func (s *server) addSession(ctx context.Context, id string, cl ktmt.ConnectionLayer) (ktmt.SessionLayer, error) {
	s.Lock()
	defer s.Unlock()

	si, ok := s.sessions.Load(id)
	if ok {
		err := si.(ktmt.SessionLayer).UpdateConnectLayer(ctx, cl)
		if err != nil {
			s.sessions.Delete(id)
			return nil, err
		}
	} else {
		ns, err := ktmt.NewSession(ctx, cl)
		if err != nil {
			return nil, err
		}
		s.sessions.Store(id, ns)
	}
	si, ok = s.sessions.Load(id)
	if !ok {
		return nil, errors.New("not exist")
	}
	ns := si.(ktmt.SessionLayer)
	if s.options.OnOnlineFunc != nil {
		err := s.options.OnOnlineFunc(id)
		if err != nil {
			s.sessions.Delete(id)
			ns.Close()
			ktmt.ERROR.Printf("[APP] add session %s OnOnlineFunc %v", id, err)
			return nil, err
		}
	}

	return ns, nil
}

func (s *server) getSession(id string) ktmt.SessionLayer {
	r, ok := s.sessions.Load(id)
	if !ok {
		return nil
	}
	return r.(ktmt.SessionLayer)
}

func (s *server) delSession(id string) {
	v, ok := s.sessions.Load(id)
	if ok {
		s.sessions.Delete(id)
		v.(ktmt.SessionLayer).Close()

		if s.options.OnOfflineFunc != nil {
			err := s.options.OnOfflineFunc(id)
			if err != nil {
				ktmt.ERROR.Printf("[APP] delSession %s OnOfflineFunc %v", id, err)
			}
		}
	}

}

func NewApp(pl ktmt.PresentationLayer, op *Options) (ktmt.ApplicationLayer, error) {
	if op.ln == nil {
		return nil, errors.New("listener is nil")
	}
	return &server{
		Router:   ktmt.NewRouter(),
		RWMutex:  sync.RWMutex{},
		sessions: sync.Map{},
		closed:   make(chan struct{}),
		options:  op,
		ln:       op.ln,
		pl:       pl,
	}, nil
}
