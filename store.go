package ktmt

import (
	"fmt"
	"strconv"

	"git.moresec.cn/zhangtian/ktmt/packets"
)

type Store interface {
	Open()
	Put(key string, message packets.ControlPacket)
	Get(key string) packets.ControlPacket
	All() []string
	Del(key string)
	Close()
	Reset()
}

const (
	inboundPrefix  = "i." // input
	outboundPrefix = "o." // output
)

// A key MUST have the form "X.[messageid]"
// where X is 'i' or 'o'
func mIDFromKey(key string) uint16 {
	s := key[2:]
	i, err := strconv.Atoi(s)
	chkerr(err)
	return uint16(i)
}

// Return true if key prefix is outbound
func IsKeyOutbound(key string) bool {
	return key[:2] == outboundPrefix
}

// Return true if key prefix is inbound
func IsKeyInbound(key string) bool {
	return key[:2] == inboundPrefix
}

// Return a string of the form "i.[id]"
func inboundKeyFromMID(id uint16) string {
	return fmt.Sprintf("%s%d", inboundPrefix, id)
}

// Return a string of the form "o.[id]"
func outboundKeyFromMID(id uint16) string {
	return fmt.Sprintf("%s%d", outboundPrefix, id)
}

func chkerr(e error) {
	if e != nil {
		panic(e)
	}
}
