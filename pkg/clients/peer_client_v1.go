package clients

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
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
	PEER_CLIENT_KEEPALIVE  time.Duration = 45 * time.Second
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
	torrentInfo *meta.MetaInfo

	AmChoking    bool
	AmInterested bool

	PeerChoking    bool
	PeerInterested bool

	PeerId string

	bencoder bencoding.Bencoder
	conn     net.Conn

	blockRequestChan  chan meta.BlockRequest
	blockResponseChan chan meta.BlockResponse
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
		log.Error(err)
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

func (c *PeerClientV1) buildKeepAliveRequestPacket() []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 0)
	return data
}

func (c *PeerClientV1) buildInterestedRequestPacket() []byte {
	data := make([]byte, 5)
	binary.BigEndian.PutUint32(data, 1)
	data[4] = byte(INTERESTED)
	return data
}

func (c *PeerClientV1) buildBlockRequestRequestPacket(block meta.BlockRequest) []byte {
	data := make([]byte, 17)
	binary.BigEndian.PutUint32(data, 13)
	data[4] = byte(REQUEST)

	// TODO: fill in the rest of the data
	binary.BigEndian.PutUint32(data[5:], uint32(block.Index))
	binary.BigEndian.PutUint32(data[9:], uint32(block.Begin))
	binary.BigEndian.PutUint32(data[13:], uint32(block.Length))
	return data
}

func (c *PeerClientV1) KeepAlive(ctx context.Context) error {
	conn := c.conn
	request := c.buildKeepAliveRequestPacket()
	// deadline, ok := ctx.Deadline()
	// if ok {
	// 	conn.SetDeadline(deadline)
	// }
	_, err := conn.Write(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *PeerClientV1) Interested(ctx context.Context) error {
	conn := c.conn
	request := c.buildInterestedRequestPacket()
	// deadline, ok := ctx.Deadline()
	// if ok {
	// 	conn.SetDeadline(deadline)
	// }
	_, err := conn.Write(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *PeerClientV1) Request(ctx context.Context, block meta.BlockRequest) error {
	log.Info("Requesting block: ", block)
	conn := c.conn
	request := c.buildBlockRequestRequestPacket(block)
	// deadline, ok := ctx.Deadline()
	// if ok {
	// 	conn.SetDeadline(deadline)
	// }
	_, err := conn.Write(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *PeerClientV1) HandShake(ctx context.Context) error {
	conn := c.conn
	request := c.buildHandshakeRequestPacket()
	// deadline, ok := ctx.Deadline()
	// if ok {
	// 	conn.SetDeadline(deadline)
	// }
	_, err := conn.Write(request)
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

// func (c *PeerClientV1) Interested(ctx context.Context) {
// 	conn, err := c.getConnection(ctx)
// 	if err != nil {
// 		log.Error(err)
// 		return
// 	}
// 	request := c.buildInterestedRequestPacket()
// }

func (c *PeerClientV1) bits(bs []byte) []int {
	r := make([]int, len(bs)*8)
	for i, b := range bs {
		for j := 0; j < 8; j++ {
			r[i*8+j] = int(b >> uint(7-j) & 0x01)
		}
	}
	return r
}

func (c *PeerClientV1) handleMessage(msg PeerMessage) {
	log.Info("handle message: ", c.peerInfo, msg.Id)
	switch msg.Id {
	case CHOKE:
		c.PeerChoking = true
	case UNCHOKE:
		c.PeerChoking = false
		log.Info("peer unchoke: ", c)
	case INTERESTED:
		c.PeerInterested = true
	case NOT_INTERESTED:
		c.PeerInterested = false
	case BITFIELD:
		availableData := c.bits(msg.Rawdata)
		log.Info("bitfield data: ", c.peerInfo, availableData)
	// case HAVE:
	// case REQUEST:
	case PIECE:
		log.Info("piece message: ", c.peerInfo, msg.Id)
		log.Info("piece message: ", c.peerInfo, binary.BigEndian.Uint32(msg.Rawdata[:4]), int(binary.BigEndian.Uint32(msg.Rawdata[4:8])))
		blockResp := meta.BlockResponse{
			BlockRequest: meta.BlockRequest{
				Index: int(binary.BigEndian.Uint32(msg.Rawdata[:4])),
				Begin: int(binary.BigEndian.Uint32(msg.Rawdata[4:8])),
				// Length: int(binary.BigEndian.Uint32(msg.Rawdata[8:12])),
			},
			Data: msg.Rawdata,
		}
		c.blockResponseChan <- blockResp
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

func (c *PeerClientV1) readData(ctx context.Context) (PeerMessage, error) {
	conn := c.conn
	reader := bufio.NewReader(conn)
	data := make([]byte, 5)

	_, err := reader.Read(data)
	if err != nil {
		return PeerMessage{}, err
	}
	length := binary.BigEndian.Uint32(data[:4])
	msgId := int8(data[4])

	if length == 0 {
		log.Info("keep alive message: ", c.peerInfo)
		return PeerMessage{}, errors.New("keep alive message")
	}

	data = make([]byte, length-1)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return PeerMessage{}, err
	}
	log.Println("received peer message with id:", msgId, c.peerInfo)
	peerMessage := PeerMessage{
		Len:     int32(length),
		Id:      msgId,
		Rawdata: data,
	}
	return peerMessage, nil
}

func (c *PeerClientV1) readLoop(ctx context.Context) {
	log.Info("read loop start for", c.peerInfo)
	for {
		select {
		case <-ctx.Done():
			log.Info("read loop done")
			return
		default:
			data, err := c.readData(ctx)
			if err != nil {
				log.WithError(err).Error("read data failed", c.peerInfo)
				return
			}
			c.handleMessage(data)
		}
	}
}

func (c *PeerClientV1) Start(ctx context.Context) {
	conn, err := c.getConnection(context.Background())
	if err != nil {
		log.Error(err)
		return
	}
	c.conn = conn

	contextWithDeadline, cancel := context.WithDeadline(ctx, time.Now().Add(PEER_CLIENT_TIMEOUT))
	defer cancel()

	err = c.HandShake(contextWithDeadline)
	if err != nil {
		log.WithError(err).Error("handshake failed", c.peerInfo)
		return
	}
	log.Info("handshake success", c.peerInfo)

	go c.readLoop(ctx)

	keepAliveTicker := time.NewTicker(PEER_CLIENT_KEEPALIVE)
	for {
		select {
		case blockRequest := <-c.blockRequestChan:
			// if peer is not interested in us, we should not request any block
			// if c.PeerChoking {
			// 	err := c.Interested(ctx)
			// 	if err != nil {
			// 		log.Error(err)
			// 	}
			// 	continue
			// }
			c.Interested(ctx)
			// c.
			// log.Info("requesting block: ", c.peerInfo, blockRequest)
			err := c.Request(ctx, blockRequest)
			if err != nil {
				log.Error(err)
			}

		case <-keepAliveTicker.C:
			err := c.KeepAlive(ctx)
			if err != nil {
				log.Error(err)
				return
			}
		}
	}

}

func (c *PeerClientV1) GetBlockRequestChan() chan<- meta.BlockRequest {
	return c.blockRequestChan
}

func (c *PeerClientV1) GetBlockResponseChan() <-chan meta.BlockResponse {
	return c.blockResponseChan
}

func NewPeerClientV1(peerId string, peerInfo PeersInfo, torrentInfo meta.MetaInfo) *PeerClientV1 {
	client := &PeerClientV1{
		peerInfo:          peerInfo,
		torrentInfo:       &torrentInfo,
		AmChoking:         true,
		AmInterested:      false,
		PeerChoking:       true,
		PeerInterested:    false,
		PeerId:            peerId,
		bencoder:          bencoding.NewBencoder(),
		blockRequestChan:  make(chan meta.BlockRequest, 5),
		blockResponseChan: make(chan meta.BlockResponse, 5),
	}
	return client
}
