package clients

import (
	"fmt"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	PSTR = "BitTorrent protocol"
)

type PeerClientV1 struct {
	peerInfo PeerInfo
}

func (c *PeerClientV1) buildHandshakeRequest(infoHash string, peerId string) []byte {
	data := make([]byte, 49+len(PSTR))
	copy(data, []byte{byte(len(PSTR))})
	copy(data[1:20], PSTR)
	copy(data[28:48], infoHash)
	copy(data[48:68], peerId)
	return data
}

func (c *PeerClientV1) HandShake(wg *sync.WaitGroup, infoHash string, peerId string) error {
	defer wg.Done()

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.peerInfo.Ip, c.peerInfo.Port))
	if err != nil {
		log.WithError(err).Error("failed to connect to peer")
		return err
	}
	defer conn.Close()

	request := c.buildHandshakeRequest(infoHash, peerId)
	_, err = conn.Write(request)
	if err != nil {
		log.WithError(err).Error("failed to send handshake request")
		return err
	}
	log.Info("sent handshake request")
	return nil
}

func NewPeerClientV1(peerInfo PeerInfo) *PeerClientV1 {
	return &PeerClientV1{
		peerInfo: peerInfo,
	}
}
