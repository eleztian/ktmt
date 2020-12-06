package mock

// Port is a fake network interface which implements the net.Port interface
type Addr struct {
	NetworkString string
	AddrString    string
}

func (a Addr) Network() string {
	return a.NetworkString
}

func (a Addr) String() string {
	return a.AddrString
}
