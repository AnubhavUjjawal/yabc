package clients

import (
	"context"
	"sync"

	"github.com/AnubhavUjjawal/yabc/pkg/meta"
)

type BitTorrentClient interface {
	RunTrackerClients(ctx context.Context, wg *sync.WaitGroup) []error
}

func NewBitTorrentClient(meta meta.MetaInfo) BitTorrentClient {
	return NewYABCBittorentClient(meta)
}
