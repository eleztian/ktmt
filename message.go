package ktmt

import (
	"context"
	"sync"

	"github.com/eleztian/ktmt/packets"
)

type Message interface {
	Topic() string
	ClientID() string
	MessageID() MId
	Qos() byte
	Duplicate() bool
	Retained() bool
	Payload() []byte
	Ack()
}

type message struct {
	topic     string
	client    string
	messageID MId
	qos       byte
	duplicate bool
	retained  bool
	payload   []byte
	once      sync.Once
	ack       func()
}

func (m *message) ClientID() string {
	return m.client
}

func (m *message) Topic() string {
	return m.topic
}

func (m *message) MessageID() MId {
	return m.messageID
}

func (m *message) Qos() byte {
	return m.qos
}

func (m *message) Duplicate() bool {
	return m.duplicate
}

func (m *message) Retained() bool {
	return m.retained
}

func (m *message) Payload() []byte {
	return m.payload
}

func (m *message) Ack() {
	m.once.Do(m.ack)
}

func MessageFromPublish(p *packets.PublishPacket, cid string, ack func()) Message {
	return &message{
		duplicate: p.Dup,
		qos:       p.Qos,
		retained:  p.Retain,
		topic:     p.TopicName,
		messageID: MId(p.MessageID),
		payload:   p.Payload,
		ack:       ack,
		client:    cid,
	}
}

func NewConnectMsg(cid string, cleanSession bool, keepalive uint16) *packets.ConnectPacket {
	m := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)

	m.CleanSession = cleanSession
	m.ClientIdentifier = cid
	m.Keepalive = keepalive

	return m
}

func AckFunc(ctx context.Context, cl ConnectionLayer, packet *packets.PublishPacket) func() {
	return func() {
		switch packet.Qos {
		case 1:
			pa := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
			pa.MessageID = packet.MessageID

			err := cl.WriteP(ctx, pa)
			if err != nil {
				ERROR.Printf("[Conn:%s] fileWrite publish[%d] ack %v", cl.ID(), pa.MessageID, err)
				cl.Close()
			}
		case 0:
			// do nothing, since there is no need to send an ack packet back
		}
	}
}
