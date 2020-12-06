package ktmt

import (
	"git.moresec.cn/zhangtian/ktmt/packets"
	"sync"
)

type memStore struct {
	sync.RWMutex
	messages map[string]packets.ControlPacket
	opened   bool
}

func NewMemoryStore() Store {
	store := &memStore{
		messages: make(map[string]packets.ControlPacket),
		opened:   false,
	}
	return store
}

func (m *memStore) Open() {
	m.Lock()
	defer m.Unlock()

	if m.opened {
		return
	}

	m.messages = make(map[string]packets.ControlPacket)
	m.opened = true
}

func (m *memStore) Put(key string, message packets.ControlPacket) {
	m.Lock()
	defer m.Unlock()

	if !m.opened {
		return
	}

	message.SetDup()
	m.messages[key] = message
}

func (m *memStore) Get(key string) packets.ControlPacket {
	m.Lock()
	defer m.Unlock()

	if !m.opened {
		return nil
	}

	return m.messages[key]
}

func (m *memStore) All() []string {
	m.Lock()
	defer m.Unlock()

	if !m.opened {
		return nil
	}

	keys := make([]string, len(m.messages))

	for k := range m.messages {
		keys = append(keys, k)
	}
	return keys
}

func (m *memStore) Del(key string) {
	m.Lock()
	defer m.Unlock()

	if !m.opened {
		return
	}

	delete(m.messages, key)
}

func (m *memStore) Close() {
	m.Lock()
	defer m.Unlock()

	m.opened = false
}

func (m *memStore) Reset() {
	m.Lock()
	defer m.Unlock()

	if !m.opened {
		return
	}

	m.messages = make(map[string]packets.ControlPacket)
}
