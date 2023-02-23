package clients

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/AnubhavUjjawal/yabc/pkg/meta"
	log "github.com/sirupsen/logrus"
)

const (
	PSTR           string = "BitTorrent protocol"
	CHOKE          int8   = 0
	UNCHOKE        int8   = 1
	INTERESTED     int8   = 2
	NOT_INTERESTED int8   = 3
	HAVE           int8   = 4
	BITFIELD       int8   = 5
	REQUEST        int8   = 6
	PIECE          int8   = 7
	CANCEL         int8   = 8
)

type PeerClientV1 struct {
	peerInfo    PeersInfo
	conn        *net.Conn
	torrentInfo *meta.MetaInfo

	AmChoking    bool
	AmInterested bool

	PeerChoking    bool
	PeerInterested bool
}

type PeerMessage struct {
	Len int32
	Id  int8

	// RawData does not include the length and id fields
	Rawdata []byte
}

func (c *PeerClientV1) buildHandshakeRequest(infoHash string, peerId string) []byte {
	data := make([]byte, 49+len(PSTR))
	copy(data, []byte{byte(len(PSTR))})
	copy(data[1:20], PSTR)
	copy(data[28:48], infoHash)
	copy(data[48:68], peerId)
	return data
}

func (c *PeerClientV1) HandShake(ctx context.Context, wg *sync.WaitGroup, infoHash string, peerId string) error {
	log.Info("handshaking init with peer: ", c.peerInfo)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", c.peerInfo.Ip, c.peerInfo.Port))
	if err != nil {
		log.WithError(err).Error("failed to connect to peer", c.peerInfo)
		return err
	}

	c.conn = &conn

	request := c.buildHandshakeRequest(infoHash, peerId)
	log.Info("sending handshake request: ", c.peerInfo)

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	deadline, ok := writeCtx.Deadline()
	if ok {
		conn.SetWriteDeadline(deadline)
	}
	_, err = conn.Write(request)
	if err != nil {
		log.WithError(err).Error("failed to send handshake request", c.peerInfo)
		return err
	}
	log.Info("sent handshake request", c.peerInfo)
	log.Info("waiting for handshake response", c.peerInfo)

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	deadline, ok = readCtx.Deadline()
	if ok {
		conn.SetReadDeadline(deadline)
	}
	data := make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		if err == io.EOF {
			log.Info("connection closed by peer")
			return nil
		}
		log.WithError(err).Error("failed to read handshake response", c.peerInfo)
		return err
	}
	if n > len(PSTR)+1 {
		lenPstr := int(data[0])
		pstr := string(data[1 : lenPstr+1])
		// log.Println("pstr: ", pstr)
		if pstr == PSTR {
			log.Info("handshake received")
		} else {
			log.Info("pstr not equal to PSTR")
			conn.Close()
		}
		// TODO: drop connection if info_hash or peer_id is not correct
	}
	return nil
}

func (c *PeerClientV1) handleClient(conn net.Conn, blockRespChan chan meta.BlockResponse) {
	log.Info("handling client: ", c.peerInfo)
	r := bufio.NewReader(conn)
	for {
		data := make([]byte, 5)
		_, err := r.Read(data)
		if err != nil {
			if err == io.EOF {
				log.Info("connection closed by peer")
				return
			}
			log.WithError(err).Error("failed to handle peer", c.peerInfo)
			return
		}
		length := binary.BigEndian.Uint32(data[:4])
		msgId := int8(data[4])
		if length == 0 {
			// log.Info("keep alive message", c.peerInfo)
			continue
		}
		data = make([]byte, length-1)
		n, err := io.ReadFull(conn, data)
		if err != nil {
			if err == io.EOF {
				log.Info("connection closed by peer")
				return
			}
			log.WithError(err).Error("failed to handle peer", c.peerInfo)
			return
		}
		log.Println("no of bytes read: ", n)
		peerMessage := PeerMessage{
			Len:     int32(length),
			Id:      msgId,
			Rawdata: data,
		}
		// log.Info("received data: ", peerMessage, c.peerInfo)
		c.handleMessage(peerMessage, blockRespChan)
	}
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

func (c *PeerClientV1) buildRequestRequest(blockReq meta.BlockRequest) []byte {
	data := make([]byte, 17)
	// log.Println(uint32(REQUEST))
	binary.BigEndian.PutUint32(data[:4], 13)
	data[4] = byte(REQUEST)
	binary.BigEndian.PutUint32(data[5:9], uint32(blockReq.Index))
	binary.BigEndian.PutUint32(data[9:13], uint32(blockReq.Begin))
	binary.BigEndian.PutUint32(data[13:17], uint32(blockReq.Length))
	// log.Println(data)
	return data
}

func (c *PeerClientV1) StartRequesting(ctx context.Context, wg *sync.WaitGroup, blockChan chan meta.BlockRequest) {
	defer wg.Done()

	for blockReq := range blockChan {
		request := c.buildRequestRequest(blockReq)
		log.Info("sending request for block: ", c.peerInfo, request, blockReq)
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		deadline, ok := writeCtx.Deadline()

		conn := *c.conn
		if ok {
			conn.SetWriteDeadline(deadline)
		}
		_, err := conn.Write(request)
		if err != nil {
			log.WithError(err).Error("failed to send request request", c.peerInfo)
			return
		}
	}

}

func (c *PeerClientV1) SendInterested(ctx context.Context, wg *sync.WaitGroup) error {
	data := make([]byte, 5)
	binary.BigEndian.PutUint32(data[:4], 1)
	data[4] = byte(INTERESTED)
	log.Info("sending interested message: ", c.peerInfo, data)
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	deadline, ok := writeCtx.Deadline()

	conn := *c.conn
	if ok {
		conn.SetWriteDeadline(deadline)
	}
	_, err := conn.Write(data)
	if err != nil {
		log.WithError(err).Error("failed to send interested message", c.peerInfo)
		return err
	}

	return nil

}

func (c *PeerClientV1) Start(ctx context.Context, wg *sync.WaitGroup, infoHash string, peerId string, blockChannel chan meta.BlockRequest, blockResp chan meta.BlockResponse) {
	log.Info("starting peer client: ", c.peerInfo)
	defer wg.Done()
	err := c.HandShake(ctx, wg, infoHash, peerId)
	if err != nil {
		log.WithError(err).Error("failed to handshake with peer")
		return
	}
	wg.Add(1)
	// c.SendInterested(ctx, wg)
	c.SendInterested(ctx, wg)
	go c.StartRequesting(ctx, wg, blockChannel)
	c.handleClient(*c.conn, blockResp)
	wg.Wait()
	// After handshake
}

func NewPeerClientV1(peerInfo PeersInfo, torrentInfo meta.MetaInfo) *PeerClientV1 {
	return &PeerClientV1{
		peerInfo:       peerInfo,
		torrentInfo:    &torrentInfo,
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
	}
}
