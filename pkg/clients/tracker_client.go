package clients

import (
	"errors"
	"net/url"
)

type TrackerClient interface {
	Announce() error
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
