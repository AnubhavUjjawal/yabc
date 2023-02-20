package clients

import (
	"errors"
	"net"
	"net/url"
)

type AnnounceData struct {
	InfoHash   string
	PeerId     string
	Downloaded int64
	Left       int64
	Uploaded   int64
	// NumWant    int32
}

type PeerInfo struct {
	Ip   net.IP
	Port uint16
}

type TrackerClient interface {
	Announce(AnnounceData) error
}

func NewTrackerClient(rawUrl string) (TrackerClient, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	switch {
	case u.Scheme == "udp":
		return NewUDPTrackerClient(u.Host), nil
	default:
		return nil, errors.New("invalid tracker url")
	}
}
