package packets

import (
	"bytes"
	"fmt"
	"io"
)

type ConnectPacket struct {
	FixedHeader
	ProtocolVersion byte

	CleanSession bool

	Keepalive uint16

	ClientIdentifier string
}

func (c *ConnectPacket) String() string {
	return fmt.Sprintf("%s protocolversion: %d cleansession: %t keepalive: %d clientId: %s", c.FixedHeader, c.ProtocolVersion, c.CleanSession, c.Keepalive, c.ClientIdentifier)
}

func (c *ConnectPacket) Write(w io.Writer) error {
	var body bytes.Buffer
	var err error

	body.WriteByte(c.ProtocolVersion)
	body.WriteByte(boolToByte(c.CleanSession) << 1)
	body.Write(encodeUint16(c.Keepalive))
	body.Write(encodeString(c.ClientIdentifier))

	c.FixedHeader.RemainingLength = body.Len()
	packet := c.FixedHeader.pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)

	return err
}

func (c *ConnectPacket) Unpack(b io.Reader) error {
	var err error
	c.ProtocolVersion, err = decodeByte(b)
	if err != nil {
		return err
	}
	options, err := decodeByte(b)
	if err != nil {
		return err
	}
	c.CleanSession = 1&(options>>1) > 0
	c.Keepalive, err = decodeUint16(b)
	if err != nil {
		return err
	}
	c.ClientIdentifier, err = decodeString(b)
	if err != nil {
		return err
	}

	return nil
}

func (c *ConnectPacket) Validate() byte {
	if len(c.ClientIdentifier) > 65535 {
		return ErrProtocolViolation
	}
	if len(c.ClientIdentifier) == 0 && !c.CleanSession {
		return ErrRefusedIDRejected
	}
	return Accepted
}

func (c *ConnectPacket) Details() Details {
	return Details{Qos: 0, MessageID: 0}
}
