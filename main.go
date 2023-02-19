// NOTE: Currently, this file is not part of the library. It is only used for testing.
package main

import (
	"log"

	"github.com/AnubhavUjjawal/yabc/pkg/clients"
)

func main() {
	// client, err := clients.NewTrackerClient("udp://tracker.opentrackr.org:1337")
	// client, err := clients.NewTrackerClient("udp://tracker.torrent.eu.org:451")
	client, err := clients.NewTrackerClient("udp://tracker.openbittorrent.com:80")
	if err != nil {
		log.Fatal(err)
	}
	err = client.Announce()
	if err != nil {
		log.Fatal(err)
	}
}
