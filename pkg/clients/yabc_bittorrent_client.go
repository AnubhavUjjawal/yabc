package clients

import (
	"context"
	"crypto/sha1"
	"sync"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

type YABCBittorentClient struct {
	trackers []TrackerClient
	meta     meta.MetaInfo

	bencoder bencoding.Bencoder
	PeerId   string
}

func (c *YABCBittorentClient) getInfoHash() string {
	infoStr, err := c.bencoder.GetRawValueFromDict(c.meta.RawData, "info")
	hasher := sha1.New()
	hasher.Write([]byte(infoStr))
	if err != nil {
		log.Fatal(err)
	}
	infoHash := hasher.Sum(nil)
	return string(infoHash)
}

func (c *YABCBittorentClient) RunTrackerClients(ctx context.Context, wg *sync.WaitGroup) []error {
	errs := make([]error, 0)
	for _, tracker := range c.trackers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.RunTrackerClient(ctx, tracker); err != nil {
				errs = append(errs, err)
			}
		}()
	}
	return errs
}

func (c *YABCBittorentClient) RunTrackerClient(ctx context.Context, tracker TrackerClient) error {
	announceRes, err := tracker.Announce(ctx, AnnounceData{
		InfoHash: c.getInfoHash(),
		PeerId:   c.PeerId,
	})
	if err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second * time.Duration(announceRes.Interval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// TODO: update other fields in AnnounceData, which will be fetched from our state
			res, err := tracker.Announce(ctx, AnnounceData{
				InfoHash: c.getInfoHash(),
				PeerId:   c.PeerId,
			})
			// We are ignoring the error in the hopes that the next announce will be successful
			if err != nil {
				log.WithError(err).Error("failed to announce to tracker")
			}
			// TODO: update our state with the new peers
			ticker.Reset(time.Second * time.Duration(res.Interval))
		case <-ctx.Done():
			log.Info("context done, stopping tracker client")
			return nil

		}
	}
}

func NewYABCBittorentClient(meta meta.MetaInfo) *YABCBittorentClient {
	trackers := []TrackerClient{}
	for _, trackerUrls := range meta.AnnounceList {
		for _, trackerUrl := range trackerUrls {
			tracker, err := NewTrackerClient(trackerUrl)
			if err != nil {
				log.WithError(err).Error("failed to create tracker client")
				continue
			}
			trackers = append(trackers, tracker)
		}
	}
	return &YABCBittorentClient{
		meta:     meta,
		PeerId:   "yabc-test",
		bencoder: bencoding.NewBencoder(),
		trackers: trackers,
	}
}
