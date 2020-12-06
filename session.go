package ktmt

import (
	"context"
	"sync"

	"github.com/eleztian/ktmt/packets"
)

type sessionLayer struct {
	mutex  sync.RWMutex
	cl     ConnectionLayer
	cancel func()
	closed bool
	wg     *sync.WaitGroup

	ids *MessageIds
	in  chan Message
	out chan *PacketAndToken
}

func NewSession(ctx context.Context, cl ConnectionLayer) (SessionLayer, error) {
	res := &sessionLayer{
		cl:  cl,
		ids: NewMessageIds(),
		in:  make(chan Message),
		out: make(chan *PacketAndToken, 10),
	}

	ctx, res.cancel = context.WithCancel(ctx)

	res.wg = &sync.WaitGroup{}
	res.wg.Add(2)

	go func() {
		defer res.wg.Done()
		read(ctx, res.in, res.ids, cl)
	}()

	go func() {
		defer res.wg.Done()
		write(ctx, res.out, cl)
	}()

	return res, nil
}

func (s *sessionLayer) UpdateConnectLayer(ctx context.Context, cl ConnectionLayer) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cl != nil {
		s.cancel()
		s.cl.Close()
		s.wg.Wait()
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		read(ctx, s.in, s.ids, cl)
	}()
	go func() {
		defer wg.Done()
		write(ctx, s.out, cl)
	}()

	s.cancel = cancel
	s.wg = wg
	s.cl = cl

	return nil
}

func (s *sessionLayer) Send(ctx context.Context, topic string, qos int, msg []byte) Token {
	t := NewToken(packets.Publish)

	pkt := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	pkt.TopicName = topic
	pkt.Qos = byte(qos)
	pkt.Payload = msg

	mid := s.ids.GetID(t)
	pkt.MessageID = uint16(mid)

	if s.isClosed() {
		t.SetError(ErrClosed)
		return t
	}

	select {
	case <-ctx.Done():
		t.SetError(ErrClosed)
		return t
	case s.out <- &PacketAndToken{P: pkt, T: t}:
	}

	return t
}

func (s *sessionLayer) In() <-chan Message {
	return s.in
}

func (s *sessionLayer) isClosed() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.closed
}

func (s *sessionLayer) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	if s.cl != nil {
		s.cl.Close()
		s.cl = nil
	}

	if s.wg != nil {
		s.cancel()
		s.wg.Wait()
	}

	if s.in != nil {
		close(s.in)
		s.in = nil
	}

	if s.out != nil {
		close(s.out)
		s.out = nil
	}

}

func read(ctx context.Context, in chan Message, ids *MessageIds, cl ConnectionLayer) {
	for {
		select {
		case pkt, ok := <-cl.Read():
			if !ok {
				return
			}
			switch msg := pkt.(type) {
			case *packets.PublishPacket:
				select {
				case <-ctx.Done():
					return
				case in <- MessageFromPublish(msg, cl.ID(), AckFunc(ctx, cl, msg)):
				}
			case *packets.PubackPacket:
				ids.GetToken(MId(msg.MessageID)).FlowComplete()
				ids.FreeID(MId(msg.MessageID))
			}
		case <-ctx.Done():
			return
		}

	}
}

func write(ctx context.Context, out chan *PacketAndToken, cl ConnectionLayer) {

	for {

		select {
		case <-ctx.Done():
			return
		case pt, ok := <-out:
			if !ok {
				return
			}
			switch msg := pt.P.(type) {
			case *packets.PublishPacket:
				err := cl.Write(ctx, pt)
				if err != nil {
					if pt.T != nil {
						pt.T.SetError(err)
					}
				}
			default:
				err := cl.WriteP(ctx, msg)
				if err != nil {
					if pt.T != nil {
						pt.T.SetError(err)
					}

				}
			}

		}
	}
}
