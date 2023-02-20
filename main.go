// NOTE: Currently, this file is not part of the library. It is only used for testing.
package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/clients"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
)

func main() {
	torrentFile := "sample_torrents/sample1.torrent"
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

	fmt.Println(token)
	announceData := clients.AnnounceData{
		InfoHash:   string(infoHash),
		PeerId:     string(token),
		Downloaded: 0,
		Uploaded:   0,
		Left:       0,
	}

	client, err := clients.NewTrackerClient("udp://tracker.opentrackr.org:1337/announce")
	// client, err := clients.NewTrackerClient("udp://tracker.torrent.eu.org:451")
	// client, err := clients.NewTrackerClient("udp://tracker.openbittorrent.com:80")
	if err != nil {
		log.Fatal(err)
	}
	err = client.Announce(announceData)
	if err != nil {
		log.Fatal(err)
	}
}
