package clients

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	MAGIC_NUMBER     int64 = 0x41727101980
	ACTION_CONNECT   int32 = 0
	DEFAULT_NUM_WANT int32 = -1

	ACTION_ANNOUNCE       = 1
	ACTION_SCRAPE         = 2
	ACTION_ERROR          = 3
	RETRIES_MAX           = 8
	TIMEOUT               = 15 * time.Second
	CONNECT_REQUEST_SIZE  = 16
	CONNECT_RESPONSE_SIZE = 16
	ANNOUNCE_REQUEST_SIZE = 98

	// This is the size of the announce response without the peers list
	ANNOUNCE_RESPONSE_STATIC_SIZE = 20
	PEER_IPV4_PORT_PAIR_SIZE      = 6
)

// https://xbtt.sourceforge.net/udp_tracker_protocol.html
// https://www.libtorrent.org/udp_tracker_protocol.html
type UDPTrackerClient struct {
	host         string
	connectionID int64

	rand *rand.Rand
	// action       int32
	// transactionID int32
}

type UDPTrackerAnnounceResponse struct {
	Action        int32
	TransactionId int32
	Interval      int32
	Leechers      int32
	Seeders       int32

	// TODO: add peers list
	Peers []PeersInfo
}

func (c *UDPTrackerClient) getConnection(ctx context.Context) (*net.UDPConn, error) {
	url := c.host
	server, err := net.ResolveUDPAddr("udp", url)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, server)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *UDPTrackerClient) buildConnectRequestPacket() []byte {
	transactionID := c.getNewTransactionID()
	action := ACTION_CONNECT

	data := make([]byte, CONNECT_REQUEST_SIZE)
	binary.BigEndian.PutUint64(data, uint64(c.connectionID))
	binary.BigEndian.PutUint32(data[8:], uint32(action))
	binary.BigEndian.PutUint32(data[12:], uint32(transactionID))
	return data
}

func (c *UDPTrackerClient) buildConnectResponsePacket() []byte {
	data := make([]byte, CONNECT_RESPONSE_SIZE)
	return data
}

func (c *UDPTrackerClient) verifyConnectResponsePacket(request []byte, response []byte) error {
	transactionIdInRes := binary.BigEndian.Uint32(response[4:8])
	transactionIdInReq := binary.BigEndian.Uint32(request[12:16])

	if transactionIdInRes != transactionIdInReq {
		return errors.New("transaction id mismatch in connect response packet")
	}
	actionInRes := binary.BigEndian.Uint32(response[0:4])
	if actionInRes != uint32(ACTION_CONNECT) {
		return fmt.Errorf("action mismatch, got: %d want: %d", actionInRes, ACTION_CONNECT)
	}
	return nil
}

func (c *UDPTrackerClient) setConnectionIdFromResponse(response []byte) {
	c.connectionID = int64(binary.BigEndian.Uint64(response[8:16]))
}

func (c *UDPTrackerClient) sendConnectRequest(ctx context.Context) ([]byte, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	request := c.buildConnectRequestPacket()
	_, err = conn.Write(request)
	if err != nil {
		return nil, err
	}

	response := c.buildConnectResponsePacket()
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, err
	}

	err = c.verifyConnectResponsePacket(request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *UDPTrackerClient) connect(ctx context.Context) error {
	log.Info("connecting to UDP tracker: ", c.host)
	numRetry := 0

	for numRetry <= RETRIES_MAX {
		timeout := time.Duration(math.Pow(2, float64(numRetry))) * TIMEOUT
		log.Debug("try and timeout: ", numRetry, timeout)

		connectContext, connectCancel := context.WithTimeout(ctx, timeout)
		defer connectCancel()

		numRetry++

		response, err := c.sendConnectRequest(connectContext)
		if err != nil {
			log.WithError(err).Warn("failed to connect to UDP tracker, retrying")
			continue
		}
		c.setConnectionIdFromResponse(response)
		log.Info("connected to UDP tracker: ", c.host, " new connection id: ", c.connectionID)
		return nil
	}
	return errors.New("failed to connect to UDP tracker, max retries exceeded")
}

func (c *UDPTrackerClient) getNewTransactionID() int32 {
	return c.rand.Int31()
}

func (c *UDPTrackerClient) buildAnnounceRequestPacket(announceData AnnounceData) []byte {
	transactionId := c.getNewTransactionID()
	data := make([]byte, ANNOUNCE_REQUEST_SIZE)
	binary.BigEndian.PutUint64(data, uint64(c.connectionID))
	binary.BigEndian.PutUint32(data[8:], uint32(ACTION_ANNOUNCE))
	binary.BigEndian.PutUint32(data[12:], uint32(transactionId))
	copy(data[16:36], announceData.InfoHash)
	copy(data[36:56], announceData.PeerId)
	binary.BigEndian.PutUint64(data[56:], uint64(announceData.Downloaded))
	binary.BigEndian.PutUint64(data[64:], uint64(announceData.Left))
	binary.BigEndian.PutUint64(data[72:], uint64(announceData.Uploaded))
	binary.BigEndian.PutUint32(data[80:], 0)
	binary.BigEndian.PutUint32(data[84:], 0)
	binary.BigEndian.PutUint32(data[88:], 0)
	numWant := DEFAULT_NUM_WANT
	binary.BigEndian.PutUint32(data[92:], uint32(numWant))
	binary.BigEndian.PutUint16(data[96:], 6767)
	return data
}

func (c *UDPTrackerClient) buildAnnounceResponsePacket() []byte {
	// data := make([]byte, ANNOUNCE_RESPONSE_STATIC_SIZE)

	// https://stackoverflow.com/a/21985400/13499618
	// It seems like we cannot call Read on a UDP datagram twice,
	// i.e. we cannot read the first 20 bytes and then read the rest of the bytes again.

	// So for now we will just allocate a large buffer and use it for all responses.
	return make([]byte, 1024*10)
}

func (c *UDPTrackerClient) verifyAnnounceResponsePacket(request []byte, response []byte) error {
	transactionIdInRes := binary.BigEndian.Uint32(response[4:8])
	transactionIdInReq := binary.BigEndian.Uint32(request[12:16])

	if transactionIdInRes != transactionIdInReq {
		return fmt.Errorf("transaction id mismatch in announce response packet, got: %d want: %d", transactionIdInRes, transactionIdInReq)
	}
	actionInRes := binary.BigEndian.Uint32(response[0:4])
	if actionInRes != uint32(ACTION_ANNOUNCE) {
		return fmt.Errorf("action mismatch, got: %d want: %d", actionInRes, ACTION_ANNOUNCE)
	}
	return nil
}

func (c *UDPTrackerClient) announceResponsePacketToStruct(response []byte) UDPTrackerAnnounceResponse {
	action := binary.BigEndian.Uint32(response[0:4])
	transactionId := binary.BigEndian.Uint32(response[4:8])
	interval := binary.BigEndian.Uint32(response[8:12])
	leechers := binary.BigEndian.Uint32(response[12:16])
	seeders := binary.BigEndian.Uint32(response[16:20])
	peers := make([]PeersInfo, 0)

	numPeers := (len(response) - ANNOUNCE_RESPONSE_STATIC_SIZE) / 6
	for i := 0; i < numPeers; i++ {
		ip := net.IP(response[20+i*6 : 24+i*6])
		peer := PeersInfo{
			Ip:   ip,
			Port: binary.BigEndian.Uint16(response[24+i*6 : 26+i*6]),
		}
		peers = append(peers, peer)
	}

	return UDPTrackerAnnounceResponse{
		Action:        int32(action),
		TransactionId: int32(transactionId),
		Interval:      int32(interval),
		Leechers:      int32(leechers),
		Seeders:       int32(seeders),
		Peers:         peers,
	}

}

func (c *UDPTrackerClient) sendAnnounceRequest(ctx context.Context, announceData AnnounceData) ([]byte, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	request := c.buildAnnounceRequestPacket(announceData)
	_, err = conn.Write(request)
	if err != nil {
		return nil, err
	}

	response := c.buildAnnounceResponsePacket()
	_, _, err = conn.ReadFromUDP(response)
	if err != nil {
		return nil, err
	}
	log.Debug("announce response: ", response)
	err = c.verifyAnnounceResponsePacket(request, response)
	if err != nil {
		return nil, err
	}

	peersCount := binary.BigEndian.Uint32(response[16:20]) + binary.BigEndian.Uint32(response[12:16])
	log.Debug("peers count: ", peersCount)
	return response[:ANNOUNCE_RESPONSE_STATIC_SIZE+int(peersCount)*6], nil
}

func (c *UDPTrackerClient) Announce(ctx context.Context, announceData AnnounceData) (AnnounceResponse, error) {
	if c.connectionID == MAGIC_NUMBER {
		err := c.connect(ctx)
		if err != nil {
			return AnnounceResponse{}, err
		}
	}

	log.Info("announcing to UDP tracker: ", c.host)
	numRetry := 0

	for numRetry <= RETRIES_MAX {
		timeout := time.Duration(math.Pow(2, float64(numRetry))) * TIMEOUT
		log.Debug("try and timeout: ", numRetry, timeout)

		announceContext, announceCancel := context.WithTimeout(ctx, timeout)
		defer announceCancel()

		numRetry++

		response, err := c.sendAnnounceRequest(announceContext, announceData)
		if err != nil {
			log.WithError(err).Warn("failed to send announce request")
			continue
		}
		announceResponse := c.announceResponsePacketToStruct(response)
		return AnnounceResponse{
			Interval: announceResponse.Interval,
			Peers:    announceResponse.Peers,
		}, nil
	}
	return AnnounceResponse{}, errors.New("failed to announce to UDP tracker")
}

func NewUDPTrackerClient(host string) TrackerClient {
	source := rand.NewSource(time.Now().UnixNano())

	// Magic number denotes that we haven't connected yet
	return &UDPTrackerClient{host, MAGIC_NUMBER, rand.New(source)}
}
