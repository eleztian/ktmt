package ktmt

import (
	"context"
	"encoding/json"
	"git.moresec.cn/zhangtian/ktmt/packets"
	"time"
)

// ConnectionLayer 连接层
type ConnectionLayer interface {
	ID() string
	Write(ctx context.Context, pkt *PacketAndToken) error
	WriteP(ctx context.Context, msg packets.ControlPacket) error
	Read() chan packets.ControlPacket
	Close()
}

// SessionLayer 会话层
type SessionLayer interface {
	Close()
	Send(ctx context.Context, topic string, qos int, msg []byte) Token
	UpdateConnectLayer(ctx context.Context, cl ConnectionLayer) error
	In() <-chan Message
}

// PresentationLayer 表示层
type PresentationLayer interface {
	Decode(src []byte, dst interface{}) (err error)
	Encode(src interface{}) (dst []byte, err error)
}

// ApplicationLayer 应用层
type ApplicationLayer interface {
	Open(ctx context.Context) error
	Close() error

	Publish(ctx context.Context, topic string, data interface{}) error
	PublishTimeout(ctx context.Context, topic string, data interface{}, timeout time.Duration) error

	PublishWithID(ctx context.Context, sid string, topic string, data interface{}) Token

	Sessions() []string

	AddRoute(topic string, callback MessageHandler)
	DeleteRoute(topic string)
	SetDefaultHandler(handler MessageHandler)

	Remove(id string)
}

type jsonPresentation struct {
}

func (j *jsonPresentation) Decode(src []byte, dst interface{}) (err error) {
	return json.Unmarshal(src, dst)
}

func (j *jsonPresentation) Encode(src interface{}) (dst []byte, err error) {
	return json.Marshal(src)
}

func NewJsonPresentationLayer() PresentationLayer {
	return &jsonPresentation{}
}
