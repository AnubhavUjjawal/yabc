package clients

import "sync"

type PeerState struct {
	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool
}

// It is important for the client to keep its peers informed as to whether or not it is interested in them.
// This state information should be kept up-to-date with each peer even when the client is choked.
// This will allow peers to know if the client will begin downloading when it is unchoked (and vice-versa).

type PeerClient interface {
	HandShake(wg *sync.WaitGroup, infoHash string, peerId string) error
}

func NewPeerClient(peerInfo PeerInfo) PeerClient {
	return NewPeerClientV1(peerInfo)
}
