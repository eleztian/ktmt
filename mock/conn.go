package mock

import (
	"io"
	"net"
	"time"
)

// End is one 'end' of a simulated connection.
type End struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

func (e End) Close() error {
	if err := e.Writer.Close(); err != nil {
		return err
	}
	if err := e.Reader.Close(); err != nil {
		return err
	}
	return nil
}

func (e End) Read(data []byte) (n int, err error)  { return e.Reader.Read(data) }
func (e End) Write(data []byte) (n int, err error) { return e.Writer.Write(data) }

func (e End) LocalAddr() net.Addr {
	return Addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

func (e End) RemoteAddr() net.Addr {
	return Addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

func (e End) SetDeadline(t time.Time) error      { return nil }
func (e End) SetReadDeadline(t time.Time) error  { return nil }
func (e End) SetWriteDeadline(t time.Time) error { return nil }

// MockConn facilitates testing by providing two connected ReadWriteClosers
// each of which can be used in place of a net.Conn
type Conn struct {
	Server *End
	Client *End
}

func NewConn() *Conn {
	// A connection consists of two pipes:
	// Client      |      Server
	//   writes   ===>  reads
	//    reads  <===   writes

	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()

	return &Conn{
		Server: &End{
			Reader: serverRead,
			Writer: serverWrite,
		},
		Client: &End{
			Reader: clientRead,
			Writer: clientWrite,
		},
	}
}

func (c *Conn) Close() error {
	if err := c.Server.Close(); err != nil {
		return err
	}
	if err := c.Client.Close(); err != nil {
		return err
	}
	return nil
}
