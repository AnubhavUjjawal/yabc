// NOTE: Currently, this file is not part of the library. It is only used for testing.
package main

import (
	"context"
	"os"
	"sync"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/clients"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

func main() {
	// log.SetLevel(log.DebugLevel)
	torrentFile := "sample_torrents/sample1.torrent"
	// torrentFile := "sample_torrents/big-buck-bunny.torrent"
	// torrentFile := "sample_torrents/cosmos-laundromat.torrent"
	// read torrent file into string
	dataBytes, err := os.ReadFile(torrentFile)
	if err != nil {
		log.Fatal(err)
	}
	// parse torrent file
	var torrentMeta meta.MetaInfo
	decoder := bencoding.NewBencoder()
	err = decoder.Unmarshal(string(dataBytes), &torrentMeta)
	if err != nil {
		log.Fatal(err)
	}

	yabc := clients.NewYABCBittorentClient(torrentMeta)
	ctx := context.Background()
	// ctx, cancel := context.WithCancel(context.Background())
	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	log.Info("Cancelling context")
	// 	cancel()
	// }()
	var wg sync.WaitGroup
	go yabc.StartListener(ctx)
	go yabc.StartPeers(ctx)
	errs := yabc.RunTrackerClients(ctx, &wg)
	if len(errs) > 0 {
		log.Info(errs)
	}
	wg.Wait()
	// infoStr, err := decoder.GetRawValueFromDict(torrentMeta.RawData, "info")
	// hasher := sha1.New()
	// hasher.Write([]byte(infoStr))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// sha1Hash := hex.EncodeToString(hasher.Sum(nil))

	// infoHash := hasher.Sum(nil)
	// token := make([]byte, 20)
	// source := rand.NewSource(time.Now().UnixNano())
	// r := rand.New(source)
	// r.Read(token)

	// log.Info(token)
	// announceData := clients.AnnounceData{
	// 	InfoHash:   string(infoHash),
	// 	PeerId:     string(token),
	// 	Downloaded: 0,
	// 	Uploaded:   0,
	// 	Left:       0,
	// }

	// client, err := clients.NewTrackerClient(data.Announce)
	// client, err := clients.NewTrackerClient("udp://tracker.torrent.eu.org:451")
	// client, err := clients.NewTrackerClient("udp://tracker.openbittorrent.com:80")
	// // client, err := clients.NewTrackerClient("udp://tracker.opentrackr.org:1337/announce")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// ctx := context.Background()
	// context, cancel := context.WithCancel(ctx)
	// yabcClient := clients.NewBitTorrentClient(client)

	// go yabcClient.RunTrackerClient(context, string(infoHash), string(token))
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	log.Info("cancelling context")
	// 	cancel()
	// 	wg.Done()
	// }()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// wg.Wait()
	// <-context.Done()

	// NOTE: use announceList if available
	// announceResponse, err := client.Announce(ctx, announceData)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // run a goroutine which sends an announce request after announceResponse.Interval seconds
	// // updates the announceResponse.Interval and repeats.
	// // log.Info("announce response: ", announceResponse)

	// if announceResponse.Peers == nil {
	// 	log.Info("no peers found")
	// } else {
	// 	log.Info("peers found: ", announceResponse.Peers)
	// }
	// var wg sync.WaitGroup
	// blockRequestChan := make(chan meta.BlockRequest)
	// pieceChan := make(chan meta.BlockResponse)
	// for _, peer := range announceResponse.Peers {
	// 	// log.Info("creating peer: ", peer)
	// 	peerClient := clients.NewPeerClient(peer, torrentMeta)
	// 	wg.Add(1)
	// 	// context, cancel := context.WithTimeout(ctx, 5*time.Second)
	// 	// defer cancel()
	// 	go peerClient.Start(
	// 		ctx,
	// 		&wg,
	// 		string(infoHash),
	// 		string(token),
	// 		blockRequestChan,
	// 		pieceChan,
	// 	)
	// 	// if err != nil {
	// 	// 	log.WithError(err).Error("failed to handshake with peer")
	// 	// }
	// 	// start a goroutine with each client
	// }
	// time.Sleep(5 * time.Second)

	// // lets make a test file.
	// // defer file.Close()
	// // Lets try to download the 1st piece
	// blockSize := int(math.Pow(2, 14))
	// // for i := 0; i < torrentMeta.Info.PieceLength; i += blockSize {
	// // 	lengthToDownload := blockSize
	// // 	if i+blockSize > torrentMeta.Info.PieceLength {
	// // 		lengthToDownload = torrentMeta.Info.PieceLength - i
	// // 	}
	// // 	blockRequestChan <- meta.BlockRequest{
	// // 		Index:  0,
	// // 		Begin:  i,
	// // 		Length: lengthToDownload,
	// // 	}
	// // }
	// go func() {
	// 	for {
	// 		blockRequestChan <- meta.BlockRequest{
	// 			Index:  0,
	// 			Begin:  0,
	// 			Length: blockSize,
	// 		}
	// 		time.Sleep(2 * time.Second)
	// 	}
	// }()
	// go func() {
	// 	for blockRes := range pieceChan {
	// 		log.Info("got piece: ", blockRes)
	// 	}
	// }()
	// log.Info("closing blockRequestChan")
	// // close(blockRequestChan)
	// wg.Wait()
}
