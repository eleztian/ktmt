package ktmt

import (
	"container/list"
	"sync"

	"git.moresec.cn/zhangtian/ktmt/packets"
)

type Customer interface {
	GetAckFunc(msg *packets.PublishPacket) func()
}

type MessageHandler func(Message) bool

type Router interface {
	// AddRoute 添加路由
	AddRoute(topic string, callback MessageHandler)
	// DeleteRoute 删除路由
	DeleteRoute(topic string)
	// SetDefaultHandler 设置默认handler.
	SetDefaultHandler(handler MessageHandler)
	// MatchAndDispatch 分发消息，并处理. order 顺序执行.
	MatchAndDispatch(messages <-chan Message, order bool)
}

type route struct {
	topic    string
	callback MessageHandler
}

func (r *route) match(topic string) bool {
	return r.topic == topic
}

type router struct {
	sync.RWMutex
	routes         *list.List
	defaultHandler MessageHandler
	messages       chan *packets.PublishPacket
}

func (r *router) AddRoute(topic string, callback MessageHandler) {
	r.Lock()
	defer r.Unlock()

	for e := r.routes.Front(); e != nil; e = e.Next() {
		if e.Value.(*route).topic == topic {
			r := e.Value.(*route)
			r.callback = callback
			return
		}
	}
	r.routes.PushBack(&route{topic: topic, callback: callback})
}

func (r *router) DeleteRoute(topic string) {
	r.Lock()
	defer r.Unlock()
	for e := r.routes.Front(); e != nil; e = e.Next() {
		if e.Value.(*route).topic == topic {
			r.routes.Remove(e)
			return
		}
	}
}

func (r *router) SetDefaultHandler(handler MessageHandler) {
	r.Lock()
	defer r.Unlock()
	r.defaultHandler = handler
}

func (r *router) MatchAndDispatch(messages <-chan Message, order bool) {
	for message := range messages {
		sent := false
		handlers := make([]MessageHandler, 0)

		r.RLock()
		for e := r.routes.Front(); e != nil; e = e.Next() {
			if e.Value.(*route).match(message.Topic()) {
				if order {
					handlers = append(handlers, e.Value.(*route).callback)
				} else {
					hd := e.Value.(*route).callback
					go func() {
						if hd(message) {
							message.Ack()
						}
					}()
				}
				sent = true
			}
		}

		if !sent { // not found callback, do default handler.
			if r.defaultHandler != nil {
				if order {
					handlers = append(handlers, r.defaultHandler)
				} else {
					go func() {
						if r.defaultHandler(message) {
							message.Ack()
						}
					}()
				}
			}
		}

		r.RUnlock()
		for _, handler := range handlers {
			if handler(message) {
				message.Ack()
			}
		}
	}
}

func NewRouter() *router {
	router := &router{routes: list.New(), messages: make(chan *packets.PublishPacket)}
	return router
}
