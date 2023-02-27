package clients

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

type YABCBittorentClient struct {
	trackers []TrackerClient
	peers    map[string]PeerClient
	meta     meta.MetaInfo

	bencoder bencoding.Bencoder
	PeerId   string

	listeningPort int16
	// blockRequestChan  chan meta.BlockRequest
	// blockResponseChan chan meta.BlockResponse

	peersChan chan PeerClient
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
		go func(t TrackerClient) {
			defer wg.Done()
			if err := c.RunTrackerClient(ctx, t); err != nil {
				errs = append(errs, err)
			}
		}(tracker)
	}
	return errs
}

func getPeerHash(peerInfo PeersInfo) string {
	return fmt.Sprintf("%s:%d", peerInfo.Ip.String(), peerInfo.Port)
}

func (c *YABCBittorentClient) addPeers(peers []PeersInfo) {
	for _, peerInfo := range peers {
		peer := NewPeerClientV1(c.PeerId, peerInfo, c.meta)

		if _, ok := c.peers[getPeerHash(peerInfo)]; ok {
			continue
		}

		c.peers[getPeerHash(peerInfo)] = peer
		c.peersChan <- peer
	}
}

func (c *YABCBittorentClient) RunTrackerClient(ctx context.Context, tracker TrackerClient) error {
	announceRes, err := tracker.Announce(ctx, AnnounceData{
		InfoHash: c.getInfoHash(),
		PeerId:   c.PeerId,
		Port:     c.listeningPort,
	})
	c.addPeers(announceRes.Peers)
	log.Info("created num peers ", len(c.peers))

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
				Port:     c.listeningPort,
			})
			// We are ignoring the error in the hopes that the next announce will be successful
			if err != nil {
				log.WithError(err).Error("failed to announce to tracker")
			}
			// TODO: update our state with the new peers
			ticker.Reset(time.Second * time.Duration(res.Interval))
			c.addPeers(res.Peers)
		case <-ctx.Done():
			log.Info("context done, stopping tracker client")
			return nil

		}
	}
}

func (c *YABCBittorentClient) handleConnection(conn net.Conn) {
	defer conn.Close()
	// read from the connection
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("received message from peer: ", string(buffer))
}

func (c *YABCBittorentClient) StartListener(ctx context.Context) {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", c.listeningPort))
	if err != nil {
		log.WithError(err).Error("failed to start listener")
		return
	}
	defer listen.Close()
	log.Info("started listening for incoming tcp connections on port ", c.listeningPort)
	for {
		select {
		case <-ctx.Done():
			log.Info("context done, stopping listener")
			return
		default:
			conn, err := listen.Accept()
			if err != nil {
				log.WithError(err).Error("failed to accept connection")
				continue
			}
			go c.handleConnection(conn)
		}
	}

}

func (c *YABCBittorentClient) ProcessBlockResponse(responseChan <-chan meta.BlockResponse) {
	for response := range responseChan {
		log.Info("received block response from peer", response)
	}
}

// Only start new peers
func (c *YABCBittorentClient) StartPeers(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				length := 1024 * 16
				for _, peer := range c.peers {
					go func(p PeerClient) {
						// log.Info("sending request to peer", p)
						requestChan := p.GetBlockRequestChan()
						// request all the blocks in the first piece
						// numBlocks := int(math.Ceil(float64(c.meta.Info.PieceLength) / float64(length)))
						// for i := 0; i < numBlocks; i++ {
						// 	requestChan <- meta.BlockRequest{
						// 		Index:  0,
						// 		Begin:  i * length,
						// 		Length: length,
						// 	}
						// }
						// log.Info("sending block request to channel", p)
						requestChan <- meta.BlockRequest{
							Index:  10,
							Begin:  length,
							Length: length,
						}
					}(peer)
				}
			case <-ctx.Done():
				log.Info("context done, stopping peers")
				return
			}
		}

	}()
	for peer := range c.peersChan {
		go func(p PeerClient) {
			p.Start(ctx)
		}(peer)
		go c.ProcessBlockResponse(peer.GetBlockResponseChan())
	}
}

func (c *YABCBittorentClient) Interested(ctx context.Context) {

}

func NewYABCBittorentClient(metainfo meta.MetaInfo) *YABCBittorentClient {
	trackers := []TrackerClient{}
	// for _, trackerUrls := range meta.AnnounceList {
	// 	for _, trackerUrl := range trackerUrls {
	// 		tracker, err := NewTrackerClient(trackerUrl)
	// 		if err != nil {
	// 			log.WithError(err).Error("failed to create tracker client")
	// 			continue
	// 		}
	// 		trackers = append(trackers, tracker)
	// 	}
	// }
	newClient, _ := NewTrackerClient("udp://tracker.opentrackr.org:1337/announce")
	trackers = append(trackers, newClient)

	return &YABCBittorentClient{
		meta: metainfo,
		// PeerId:        uuid.New().String(),
		PeerId:        "new-client-0001",
		bencoder:      bencoding.NewBencoder(),
		trackers:      trackers,
		peers:         make(map[string]PeerClient),
		listeningPort: 6881,

		// todo: make size configurable to allow request pipelining
		// blockRequestChan:  make(chan meta.BlockRequest),
		// blockResponseChan: make(chan meta.BlockResponse),
		peersChan: make(chan PeerClient),
	}
}
