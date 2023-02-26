package clients

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

const (
	PSTR                   string        = "BitTorrent protocol"
	LEN_PSTR               int           = len(PSTR)
	HANDSHAKE_REQUEST_SIZE int           = 49 + LEN_PSTR
	PEER_CLIENT_TIMEOUT    time.Duration = 5 * time.Second
	CHOKE                  int8          = 0
	UNCHOKE                int8          = 1
	INTERESTED             int8          = 2
	NOT_INTERESTED         int8          = 3
	HAVE                   int8          = 4
	BITFIELD               int8          = 5
	REQUEST                int8          = 6
	PIECE                  int8          = 7
	CANCEL                 int8          = 8
)

type PeerClientV1 struct {
	peerInfo    PeersInfo
	conn        *net.Conn
	torrentInfo *meta.MetaInfo

	AmChoking    bool
	AmInterested bool

	PeerChoking    bool
	PeerInterested bool

	PeerId string

	bencoder bencoding.Bencoder
}

type PeerMessage struct {
	Len int32
	Id  int8

	// RawData does not include the length and id fields
	Rawdata []byte
}

func (c *PeerClientV1) getConnection(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", c.peerInfo.Ip, c.peerInfo.Port))
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *PeerClientV1) buildHandshakeRequestPacket() []byte {
	data := make([]byte, HANDSHAKE_REQUEST_SIZE)
	copy(data, []byte{byte(LEN_PSTR)})
	copy(data[1:20], PSTR)
	copy(data[28:48], []byte(c.getInfoHash()))
	copy(data[48:68], []byte(c.PeerId))
	return data
}

func (c *PeerClientV1) buildHandshakeResponsePacket() []byte {
	data := make([]byte, HANDSHAKE_REQUEST_SIZE)
	return data
}

func (c *PeerClientV1) verifyHandshakeResponsePacket(request []byte, response []byte) error {
	// TODO: drop connection if info_hash or peer_id is not correct

	lenPstr := int(response[0])
	pstr := string(response[1 : lenPstr+1])
	if pstr != PSTR {
		log.Error("pstr not equal to PSTR")
	}
	return nil
}

func (c *PeerClientV1) HandShake(ctx context.Context) error {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return err
	}

	request := c.buildHandshakeRequestPacket()
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}
	_, err = conn.Write(request)
	if err != nil {
		return err
	}

	response := c.buildHandshakeResponsePacket()
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return err
	}

	err = c.verifyHandshakeResponsePacket(request, response)
	if err != nil {
		conn.Close()
		return err
	}
	return nil
}

func (c *PeerClientV1) bits(bs []byte) []int {
	r := make([]int, len(bs)*8)
	for i, b := range bs {
		for j := 0; j < 8; j++ {
			r[i*8+j] = int(b >> uint(7-j) & 0x01)
		}
	}
	return r
}

func (c *PeerClientV1) handleMessage(msg PeerMessage, blockRespChan chan meta.BlockResponse) {
	switch msg.Id {
	case CHOKE:
		c.PeerChoking = true
	case UNCHOKE:
		c.PeerChoking = false
	case INTERESTED:
		c.PeerInterested = true
	case NOT_INTERESTED:
		c.PeerInterested = false
	case BITFIELD:
		log.Info("bitfield message: ", c)
		// availableData := c.bits(msg.Rawdata)
		// log.Info("available data: ", c, availableData)
	case HAVE:
		log.Info("have message: ", c, msg)
	case REQUEST:
		log.Info("request message: ", c, msg)
	case PIECE:
		// log.Info("piece message: ", c, msg)
		blockResp := meta.BlockResponse{
			BlockRequest: meta.BlockRequest{
				Index: int(binary.BigEndian.Uint32(msg.Rawdata[:4])),
				Begin: int(binary.BigEndian.Uint32(msg.Rawdata[4:8])),
				// Length: int(binary.BigEndian.Uint32(msg.Rawdata[8:12])),
			},
			Data: msg.Rawdata[8:],
		}
		log.Info("piece message: ", c, msg.Len)
		blockRespChan <- blockResp
	}
}

func (c *PeerClientV1) getInfoHash() string {
	infoStr, err := c.bencoder.GetRawValueFromDict(c.torrentInfo.RawData, "info")
	hasher := sha1.New()
	hasher.Write([]byte(infoStr))
	if err != nil {
		log.Fatal(err)
	}
	infoHash := hasher.Sum(nil)
	return string(infoHash)
}

func NewPeerClientV1(peerId string, peerInfo PeersInfo, torrentInfo meta.MetaInfo) *PeerClientV1 {
	return &PeerClientV1{
		peerInfo:       peerInfo,
		torrentInfo:    &torrentInfo,
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
		PeerId:         peerId,
		bencoder:       bencoding.NewBencoder(),
	}
}
