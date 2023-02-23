package clients

import (
	"context"
	"sync"

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
	HandShake(cxt context.Context, wg *sync.WaitGroup, infoHash string, peerId string) error
	Start(cxt context.Context, wg *sync.WaitGroup, infoHash string, peerId string, blockChannel chan meta.BlockRequest, blockResp chan meta.BlockResponse)
}

func NewPeerClient(peerInfo PeersInfo, torrentInfo meta.MetaInfo) PeerClient {
	return NewPeerClientV1(peerInfo, torrentInfo)
}
