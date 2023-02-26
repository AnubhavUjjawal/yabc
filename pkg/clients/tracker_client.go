package clients

import (
	"context"
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
	Port       int16
	// NumWant    int32
}

type PeersInfo struct {
	Ip   net.IP
	Port uint16
}

type AnnounceResponse struct {
	// Interval in seconds that the client should wait between sending regular requests to the tracker
	Interval int32
	Peers    []PeersInfo
}

type TrackerClient interface {
	Announce(context.Context, AnnounceData) (AnnounceResponse, error)
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
