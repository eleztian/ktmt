package ktmt

import (
	"fmt"
	"sync"
	"time"
)

type MId uint16

const (
	midMin MId = 1
	//midMax MId = 65535
)

type MessageIds struct {
	sync.RWMutex
	index map[MId]TokenCompleter
}

func NewMessageIds() *MessageIds {
	return &MessageIds{index: map[MId]TokenCompleter{}}
}

func (m *MessageIds) CleanUp() {
	m.Lock()
	defer m.Unlock()
	for _, token := range m.index {
		switch token.(type) {
		case *PublishToken:
			token.SetError(fmt.Errorf("connection lost before Publish completed"))
		case nil:
			continue
		}
		if token != nil {
			token.FlowComplete()
		}
	}
	m.index = make(map[MId]TokenCompleter)
}

func (m *MessageIds) FreeID(id MId) {
	m.Lock()
	delete(m.index, id)
	m.Unlock()
}

func (m *MessageIds) ClaimID(token TokenCompleter, id MId) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.index[id]; !ok {
		m.index[id] = token
	} else { // 等待完成后占用
		old := m.index[id]
		old.FlowComplete()
		m.index[id] = token
	}
}

func (m *MessageIds) GetID(t TokenCompleter) MId {
	m.Lock()
	defer m.Unlock()
	// uint16 最大值为 midMax(65535)
	for i := midMin; i != 0; i++ {
		if _, ok := m.index[i]; !ok {
			m.index[i] = t
			return i
		}
	}
	return 0
}

func (m *MessageIds) GetToken(id MId) TokenCompleter {
	m.RLock()
	defer m.RUnlock()
	if token, ok := m.index[id]; ok {
		return token
	}
	return &DummyToken{id: id}
}

type DummyToken struct {
	id MId
}

func (d *DummyToken) Wait() bool {
	return true
}

func (d *DummyToken) WaitTimeout(time.Duration) bool {
	return true
}

func (d *DummyToken) FlowComplete() {
}

func (d *DummyToken) Error() error {
	return nil
}

func (d *DummyToken) SetError(error) {}
