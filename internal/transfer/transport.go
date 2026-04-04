package transfer

type Transport interface {
	SendPeer(payload []byte) error
	RecvPeer() ([]byte, error)
	Close()
}
