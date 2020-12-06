package ktmt

import (
	"context"
	"net"
	"sync"
	"time"

	"git.moresec.cn/zhangtian/ktmt/packets"
)

type OnCloseCallback func(id string)

type ConnConfig struct {
	Version      byte
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CID          string
	Keepalive    uint16 // s
	CleanSession bool
	CloseCB      OnCloseCallback
}

// Connect 连接到服务端
func Connect(conn net.Conn, cf *ConnConfig) (MConn, error) {

	// send connect
	rp := NewConnectMsg(cf.CID, cf.CleanSession, cf.Keepalive*3)
	rp.ProtocolVersion = cf.Version

	if cf.WriteTimeout != 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(cf.WriteTimeout))
	}
	err := rp.Write(conn)
	if err != nil {
		return nil, err
	}
	if cf.WriteTimeout != 0 {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	// read connack
	if cf.ReadTimeout != 0 {
		_ = conn.SetReadDeadline(time.Now().Add(cf.ReadTimeout))
	}
	msg, err := packets.ReadPacket(conn)
	if err != nil {
		return nil, err
	}
	if cf.ReadTimeout != 0 {
		_ = conn.SetReadDeadline(time.Time{})
	}

	ack, ok := msg.(*packets.ConnackPacket)
	if !ok {
		return nil, err
	}

	// check rc
	if ack.ReturnCode != packets.Accepted {
		return nil, packets.ConnErrors[ack.ReturnCode]
	}

	return NewMConn(conn, cf), nil
}

// Accept 接收客户端连接
func Accept(conn net.Conn, cf *ConnConfig) (MConn, error) {
	// read conn
	if cf.ReadTimeout != 0 {
		_ = conn.SetReadDeadline(time.Now().Add(cf.ReadTimeout))
	}
	msg, err := packets.ReadPacket(conn)
	if err != nil {
		return nil, err
	}
	if cf.ReadTimeout != 0 {
		_ = conn.SetReadDeadline(time.Time{})
	}

	connReq, ok := msg.(*packets.ConnectPacket)
	if !ok {
		return nil, err
	}
	cf.Version = connReq.ProtocolVersion
	cf.CID = connReq.ClientIdentifier
	cf.CleanSession = connReq.CleanSession
	cf.Keepalive = connReq.Keepalive

	// send connack
	ack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)

	if cf.WriteTimeout != 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(cf.WriteTimeout))
	}
	err = ack.Write(conn)
	if err != nil {
		return nil, err
	}
	if cf.WriteTimeout != 0 {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	return NewMConn(conn, cf), nil
}

type MConn interface {
	net.Conn
	ID() string
	Heartbeat() time.Duration
	CleanSession() bool
}

func NewMConn(conn net.Conn, options *ConnConfig) MConn {
	return &mConn{
		Conn:         conn,
		id:           options.CID,
		hbInterval:   time.Duration(options.Keepalive) * time.Second,
		cleanSession: options.CleanSession,
		rt:           options.ReadTimeout,
		wt:           options.WriteTimeout,
		cb:           options.CloseCB,
	}
}

type mConn struct {
	net.Conn
	id           string
	hbInterval   time.Duration
	cleanSession bool
	cb           func(id string)
	rt           time.Duration
	wt           time.Duration
}

func (c mConn) Heartbeat() time.Duration {
	return c.hbInterval
}
func (c mConn) ID() string {
	return c.id
}
func (c mConn) CleanSession() bool {
	return c.cleanSession
}
func (c *mConn) Read(b []byte) (n int, err error) {
	if c.rt != 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.rt))
	}

	n, err = c.Conn.Read(b)
	if c.rt != 0 {
		_ = c.Conn.SetReadDeadline(time.Time{})
	}

	return
}

func (c *mConn) Write(b []byte) (n int, err error) {
	if c.wt != 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.wt))
	}

	n, err = c.Conn.Write(b)
	if c.wt != 0 {
		_ = c.Conn.SetWriteDeadline(time.Time{})
	}

	return
}

func (c *mConn) Close() error {
	err := c.Conn.Close()
	if err == nil && c.cb != nil {
		c.cb(c.id)
	}
	return err
}

func NewConnectionKeeper(ctx context.Context, connFactory func(context.Context) MConn, sendPing bool) ConnectionLayer {

	res := &connKeep{
		sendPing: sendPing,
		status:   &status{},
	}

	res.loop(ctx, connFactory)

	return res
}

type connKeep struct {
	sendPing bool

	*status

	ctx    context.Context
	cancel func()

	readCh   chan packets.ControlPacket
	writePCh chan packets.ControlPacket
	writeCh  chan *PacketAndToken

	stop chan struct{}

	cid string
}

func (c *connKeep) loop(ctx context.Context, connFactory func(ctx context.Context) MConn) {
	c.ctx, c.cancel = context.WithCancel(ctx)

	conn := connFactory(c.ctx)
	if conn == nil {
		return
	}

	c.cid = conn.ID()
	c.stop = make(chan struct{})
	c.writePCh = make(chan packets.ControlPacket)
	c.writeCh = make(chan *PacketAndToken)
	c.readCh = make(chan packets.ControlPacket)

	go func() {
		defer c.cancel()

		defer close(c.stop)
		defer close(c.writePCh)
		defer close(c.writeCh)
		defer close(c.readCh)

		for conn != nil {
			c.startConn(conn)
			if c.status.isClosed() {
				break
			}
			// 指数退避
			conn = connFactory(c.ctx)
		}
	}()

}

func (c *connKeep) startConn(conn MConn) {
	defer conn.Close()

	c.status.init()

	id := conn.ID()
	if c.cid != "" && id != c.cid {
		ERROR.Printf("[Conn:%s] Client id must be %s", id, c.cid)
		return
	} else {
		c.cid = id
	}

	pingRespCh := make(chan packets.ControlPacket)
	defer close(pingRespCh)

	ctxTask, cancelTask := context.WithCancel(c.ctx)
	defer cancelTask()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancelTask()
		c.read(ctxTask, conn, pingRespCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancelTask()
		defer conn.Close()
		c.write(ctxTask, conn)
	}()

	hb := conn.Heartbeat()
	if hb > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancelTask()
			c.keepalive(ctxTask, hb, pingRespCh)
		}()
	}

	wg.Wait()
}

func (c *connKeep) read(ctx context.Context, conn net.Conn, pingRespCh chan<- packets.ControlPacket) {
	for {
		pkt, err := packets.ReadPacket(conn)
		if err != nil {
			if IsNetTimeout(err) {
				time.Sleep(time.Microsecond * 20)
				continue
			}
			break
		}
		c.status.recv()

		INFO.Printf("[Conn:%s] Recv: %s", c.cid, pkt.String())

		switch msg := pkt.(type) {
		case *packets.PingreqPacket:
			go func() {
				rsp := packets.NewControlPacket(packets.Pingresp)
				select {
				case <-ctx.Done():
					return
				case c.writePCh <- rsp:
				}
			}()

		case *packets.PingrespPacket:
			select {
			case <-ctx.Done():
				return
			case pingRespCh <- pkt:
			}

		case *packets.DisconnectPacket:
			return
		default:
			select {
			case <-ctx.Done():
				return
			case c.readCh <- msg:
			}
		}
	}

	return
}

func (c *connKeep) write(ctx context.Context, conn net.Conn) {
	var err error
	var e, ep = false, false
	for {
		if e && ep {
			break
		}
		select {
		case pkt, ok := <-c.writeCh:
			if !ok {
				e = true
				break
			}
			err = pkt.P.Write(conn)
			if err != nil {
				if pkt.T != nil {
					pkt.T.SetError(err)
				}
				if IsNetTimeout(err) {
					time.Sleep(time.Microsecond * 20)
					break
				}
				return
			}

			INFO.Printf("[Conn:%s] Send: %s", c.cid, pkt.P.String())
			c.status.send()
		case pkt, ok := <-c.writePCh:
			if !ok {
				ep = true
				break
			}
			err = pkt.Write(conn)
			if err != nil {
				if IsNetTimeout(err) {
					time.Sleep(time.Microsecond * 20)
					break
				}
				return
			}
			DEBUG.Printf("[Conn:%s] SendP: %s", c.cid, pkt.String())
			c.status.send()
		case <-ctx.Done():
			return
		}
	}
}

func (c *connKeep) keepalive(ctx context.Context, hbInterval time.Duration, respCh <-chan packets.ControlPacket) {
	tk := time.NewTicker(hbInterval)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			if c.sendPing {
				pkt := packets.NewControlPacket(packets.Pingreq)
				select {
				case <-ctx.Done():
					return
				case c.writePCh <- pkt:
					// 超时等待pingresp
					ctxR, cancelR := context.WithTimeout(ctx, hbInterval)
					select {
					case <-ctxR.Done():
						cancelR()
						ERROR.Printf("[Conn:%s] wait ping resp timeout(%v)", c.cid, hbInterval)
						return
					case <-respCh:
						cancelR()
					}
				}
			} else {
				if c.expire(hbInterval) {
					ERROR.Printf("[Conn:%s] heart beat expire", c.cid)
					return
				}
			}
		}
	}

}

func (c *connKeep) Write(ctx context.Context, pkt *PacketAndToken) error {
	defer func() {
		recover()
	}()
	select {
	case <-ctx.Done():
		return ErrCanceled
	case <-c.ctx.Done():
		return ErrClosed
	case c.writeCh <- pkt:

	}

	return nil
}

func (c *connKeep) WriteP(ctx context.Context, pkt packets.ControlPacket) error {
	defer func() {
		recover()
	}()
	select {
	case <-ctx.Done():
		return ErrCanceled
	case <-c.ctx.Done():
		return ErrClosed
	case c.writePCh <- pkt:

	}

	return nil
}

func (c *connKeep) Read() chan packets.ControlPacket {
	return c.readCh
}

func (c *connKeep) ID() string {
	return c.cid
}

func (c *connKeep) Close() {
	if c.sendPing {
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		_ = c.WriteP(ctx, packets.NewControlPacket(packets.Disconnect))
		cancel()
	}

	c.cancel()
	c.status.close()

	// 等待退出
	<-c.stop
}

type status struct {
	lastRecv time.Time
	lastSend time.Time
	closed   bool
	lock     sync.RWMutex
}

func (s *status) init() {
	tn := time.Now()
	s.lock.Lock()
	s.lastRecv = tn
	s.lastSend = tn
	s.lock.Unlock()
}

func (s *status) recv() {
	s.lock.Lock()
	s.lastRecv = time.Now()
	s.lock.Unlock()
}

func (s *status) send() {
	s.lock.Lock()
	s.lastSend = time.Now()
	s.lock.Unlock()
}

func (s *status) isClosed() bool {
	s.lock.RLock()
	res := s.closed
	s.lock.RUnlock()
	return res
}

func (s *status) close() {
	s.lock.Lock()
	s.closed = true
	s.lock.Unlock()
}

func (s *status) expire(interval time.Duration) bool {
	tn := time.Now()

	s.lock.RLock()
	ls := s.lastSend
	lr := s.lastRecv
	s.lock.RUnlock()

	if tn.Sub(ls) > interval || tn.Sub(lr) > interval {
		return true
	}

	return false
}
