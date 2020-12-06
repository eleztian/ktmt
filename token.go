package ktmt

import (
	"sync"
	"time"

	"github.com/eleztian/ktmt/packets"
)

type PacketAndToken struct {
	P packets.ControlPacket
	T TokenCompleter
}

type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

type TokenErrorSetter interface {
	SetError(error)
}

type TokenCompleter interface {
	Token
	TokenErrorSetter
	FlowComplete()
}

type baseToken struct {
	m        sync.RWMutex
	complete chan struct{}
	err      error
}

func (b *baseToken) Wait() bool {
	<-b.complete
	return true
}

func (b *baseToken) WaitTimeout(d time.Duration) bool {
	timer := time.NewTimer(d)
	select {
	case <-b.complete:
		if !timer.Stop() {
			<-timer.C
		}
		return true
	case <-timer.C:
	}

	return false
}

func (b *baseToken) FlowComplete() {
	select {
	case <-b.complete:
	default:
		close(b.complete)
	}
}

func (b *baseToken) Error() error {
	b.m.RLock()
	defer b.m.RUnlock()

	return b.err
}

func (b *baseToken) SetError(e error) {
	b.m.Lock()
	defer b.m.Unlock()

	b.err = e
	b.FlowComplete()
}

// =======
type ConnectToken struct {
	baseToken
	returnCode     byte
	sessionPresent bool
}

func (c *ConnectToken) SetReturnCode(code byte) {
	c.m.Lock()
	defer c.m.Unlock()

	c.returnCode = code
}

func (c *ConnectToken) SetSessionPresent(sp bool) {
	c.m.Lock()
	defer c.m.Unlock()

	c.sessionPresent = sp
}

func (c *ConnectToken) ReturnCode() byte {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.returnCode
}

func (c *ConnectToken) SessionPresent() bool {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.sessionPresent
}

// ==
type PublishToken struct {
	baseToken
	messageID MId
}

func (p *PublishToken) MessageID() MId {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.messageID
}

func (p *PublishToken) SetMessageID(id MId) {
	p.m.Lock()
	defer p.m.Unlock()

	p.messageID = id
}

// =====
type DisconnectToken struct {
	baseToken
}

func NewToken(tType byte) TokenCompleter {
	switch tType {
	case packets.Connect:
		return &ConnectToken{baseToken: baseToken{complete: make(chan struct{})}}
	case packets.Publish:
		return &PublishToken{baseToken: baseToken{complete: make(chan struct{})}}
	case packets.Disconnect:
		return &DisconnectToken{baseToken: baseToken{complete: make(chan struct{})}}
	}
	return nil
}

type ErrorToken struct {
	err error
}

func (e *ErrorToken) Wait() bool {
	return true
}

func (e *ErrorToken) WaitTimeout(time.Duration) bool {
	return true
}

func (e *ErrorToken) Error() error {
	return e.err
}

func NewErrToken(err error) Token {
	return &ErrorToken{err: err}
}
