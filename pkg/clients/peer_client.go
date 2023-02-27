package clients

import (
	"context"

	"github.com/AnubhavUjjawal/yabc/pkg/meta"
)

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
	HandShake(context.Context) error
	Start(context.Context)

	GetBlockRequestChan() chan<- meta.BlockRequest
	GetBlockResponseChan() <-chan meta.BlockResponse
}

func NewPeerClient(peerId string, peerInfo PeersInfo, torrentInfo meta.MetaInfo) PeerClient {
	return NewPeerClientV1(peerId, peerInfo, torrentInfo)
}
