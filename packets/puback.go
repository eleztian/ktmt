package packets

import (
	"fmt"
	"io"
)

type PubackPacket struct {
	FixedHeader
	MessageID uint16
}

func (pa *PubackPacket) String() string {
	return fmt.Sprintf("%s MessageID: %d", pa.FixedHeader, pa.MessageID)
}

func (pa *PubackPacket) Write(w io.Writer) error {
	var err error
	pa.FixedHeader.RemainingLength = 2
	packet := pa.FixedHeader.pack()
	packet.Write(encodeUint16(pa.MessageID))
	_, err = packet.WriteTo(w)

	return err
}

//Unpack decodes the details of a ControlPacket after the fixed
//header has been read
func (pa *PubackPacket) Unpack(b io.Reader) error {
	var err error
	pa.MessageID, err = decodeUint16(b)

	return err
}

//Details returns a Details struct containing the Qos and
//MessageID of this ControlPacket
func (pa *PubackPacket) Details() Details {
	return Details{Qos: pa.Qos, MessageID: pa.MessageID}
}