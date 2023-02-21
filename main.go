// NOTE: Currently, this file is not part of the library. It is only used for testing.
package main

import (
	"crypto/sha1"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/clients"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

func main() {
	// torrentFile := "sample_torrents/big-buck-bunny.torrent"
	torrentFile := "sample_torrents/cosmos-laundromat.torrent"
	// read torrent file into string
	dataBytes, err := os.ReadFile(torrentFile)
	if err != nil {
		log.Fatal(err)
	}
	// parse torrent file
	var data meta.MetaInfo
	decoder := bencoding.NewBencoder()
	err = decoder.Unmarshal(string(dataBytes), &data)
	if err != nil {
		log.Fatal(err)
	}
	infoStr, err := decoder.GetRawValueFromDict(data.RawData, "info")
	hasher := sha1.New()
	hasher.Write([]byte(infoStr))
	if err != nil {
		log.Fatal(err)
	}
	// sha1Hash := hex.EncodeToString(hasher.Sum(nil))

	infoHash := hasher.Sum(nil)
	token := make([]byte, 20)
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	r.Read(token)

	log.Info(token)
	announceData := clients.AnnounceData{
		InfoHash:   string(infoHash),
		PeerId:     string(token),
		Downloaded: 0,
		Uploaded:   0,
		Left:       0,
	}

	// client, err := clients.NewTrackerClient(data.Announce)
	// client, err := clients.NewTrackerClient("udp://tracker.torrent.eu.org:451")
	// client, err := clients.NewTrackerClient("udp://tracker.openbittorrent.com:80")
	client, err := clients.NewTrackerClient("udp://tracker.opentrackr.org:1337/announce")
	if err != nil {
		log.Fatal(err)
	}

	// NOTE: use announceList if available
	announceResponse, err := client.Announce(announceData)
	if err != nil {
		log.Fatal(err)
	}

	// run a goroutine which sends an announce request after announceResponse.Interval seconds
	// updates the announceResponse.Interval and repeats.
	// log.Info("announce response: ", announceResponse)

	if announceResponse.Peers == nil {
		log.Info("no peers found")
	} else {
		log.Info("peers found: ", announceResponse.Peers)
	}
	var wg sync.WaitGroup
	for _, peer := range announceResponse.Peers {
		peerClient := clients.NewPeerClient(peer)
		wg.Add(1)
		go peerClient.HandShake(&wg, string(infoHash), string(token))
		// if err != nil {
		// 	log.WithError(err).Error("failed to handshake with peer")
		// }
		// start a goroutine with each client
	}
	wg.Wait()
}
