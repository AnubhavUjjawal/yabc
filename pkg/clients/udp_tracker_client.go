package clients

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	MAGIC_NUMBER    int64 = 0x41727101980
	ACTION_CONNECT  int32 = 0
	ACTION_ANNOUNCE       = 1
	ACTION_SCRAPE         = 2
	ACTION_ERROR          = 3
	RETRIES_MAX           = 8
)

// https://xbtt.sourceforge.net/udp_tracker_protocol.html
// https://www.libtorrent.org/udp_tracker_protocol.html
type UDPTrackerClient struct {
	host         string
	connectionID int64
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
	Peers []PeerInfo
}

func (c *UDPTrackerClient) getConnection() (*net.UDPConn, error) {
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

	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data, uint64(c.connectionID))
	binary.BigEndian.PutUint32(data[8:], uint32(action))
	binary.BigEndian.PutUint32(data[12:], uint32(transactionID))
	log.Info("initial connect data ", c.connectionID, action, transactionID)
	return data
}

func (c *UDPTrackerClient) verifyConnectResponsePacket(request []byte, response []byte) error {
	transactionIdInRes := binary.BigEndian.Uint32(response[4:8])
	transactionIdInReq := binary.BigEndian.Uint32(request[12:16])

	if transactionIdInRes != transactionIdInReq {
		log.Warn("transaction ID mismatch ", transactionIdInReq, " ", transactionIdInRes)
		return errors.New("transaction ID mismatch")
	}
	actionInRes := binary.BigEndian.Uint32(response[0:4])
	if actionInRes != uint32(ACTION_CONNECT) {
		log.Warn("action mismatch, retrying ", actionInRes, " ", ACTION_CONNECT)
		return errors.New("action mismatch")
	}
	return nil
}

func (c *UDPTrackerClient) setConnectionID(response []byte) {
	c.connectionID = int64(binary.BigEndian.Uint64(response[8:16]))
}

// TODO: refactor this method
func (c *UDPTrackerClient) sendConnectRequest(readTimeout time.Duration) ([]byte, error) {
	request := c.buildConnectRequestPacket()
	conn, err := c.getConnection()
	if err != nil {
		log.Warn("failed to get connection: ", err)
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write(request)
	if err != nil {
		log.Warn("failed to write to UDP tracker: ", err)
		return nil, err
	}

	err = conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		log.Warn("failed to set read deadline: ", err)
		return nil, err
	}

	response := make([]byte, 16)
	n, _, err := conn.ReadFromUDP(response)
	if err != nil {
		log.Warn("failed to read from UDP tracker: ", err)
		return nil, err
	}
	if n < 16 {
		log.Warn("received less than 16 bytes: ", n)
		return nil, errors.New("received less than 16 bytes")
	}
	err = c.verifyConnectResponsePacket(request, response)
	if err != nil {
		log.Warn("failed to verify connect response packet: ", err)
		return nil, err
	}
	return response, nil
}

func (c *UDPTrackerClient) connect() error {
	log.Info("connecting to UDP tracker: ", c.host)
	tryNum := 0

	for tryNum <= RETRIES_MAX {
		log.Info("try number: ", tryNum)
		readTimeout := time.Duration(math.Pow(2, float64(tryNum))*15) * time.Second
		log.Info("read timeout: ", readTimeout)

		// increasing try number before we forget
		tryNum++

		response, err := c.sendConnectRequest(readTimeout)
		if err != nil {
			log.Warn("failed to send connect request: ", err)
			continue
		}
		c.setConnectionID(response)
		return nil
	}
	return errors.New("failed to connect to UDP tracker, max retries exceeded")
}

func (c *UDPTrackerClient) getNewTransactionID() int32 {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	return r.Int31()
}

func (c *UDPTrackerClient) buildAnnounceRequestPacket(announceData AnnounceData) []byte {
	transactionId := c.getNewTransactionID()
	log.Info("new transaction ID: ", transactionId)
	data := make([]byte, 98)
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
	numWant := -1
	binary.BigEndian.PutUint32(data[92:], uint32(numWant))
	binary.BigEndian.PutUint16(data[96:], 6767)
	return data
}

func (c *UDPTrackerClient) verifyAnnounceResponsePacket(request []byte, response []byte) error {
	transactionIdInRes := binary.BigEndian.Uint32(response[4:8])
	transactionIdInReq := binary.BigEndian.Uint32(request[12:16])

	if transactionIdInRes != transactionIdInReq {
		log.Warn("transaction ID mismatch ", transactionIdInReq, " ", transactionIdInRes)
		return errors.New("transaction ID mismatch")
	}
	actionInRes := binary.BigEndian.Uint32(response[0:4])
	if actionInRes != uint32(ACTION_ANNOUNCE) {
		log.Warn("action mismatch, retrying ", actionInRes, " ", ACTION_ANNOUNCE)
		return errors.New("action mismatch")
	}
	return nil
}

func (c *UDPTrackerClient) announceResponsePacketToStruct(response []byte, numPeers int) UDPTrackerAnnounceResponse {
	action := binary.BigEndian.Uint32(response[0:4])
	transactionId := binary.BigEndian.Uint32(response[4:8])
	interval := binary.BigEndian.Uint32(response[8:12])
	leechers := binary.BigEndian.Uint32(response[12:16])
	seeders := binary.BigEndian.Uint32(response[16:20])
	peers := make([]PeerInfo, 0)
	for i := 0; i < numPeers; i++ {
		ip := net.IP(response[20+i*6 : 24+i*6])
		peer := PeerInfo{
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

func (c *UDPTrackerClient) Announce(announceData AnnounceData) error {
	log.Info("announcing to UDP tracker: ", c.host)
	if c.connectionID == MAGIC_NUMBER {
		c.connect()
	}
	log.Info("connection ID:", c.connectionID)
	request := c.buildAnnounceRequestPacket(announceData)
	tryNum := 0

	for tryNum <= RETRIES_MAX {
		log.Info("try number: ", tryNum)
		readTimeout := time.Duration(math.Pow(2, float64(tryNum))*15) * time.Second
		log.Info("read timeout: ", readTimeout)

		// increasing try number before we forget
		tryNum++

		conn, err := c.getConnection()
		if err != nil {
			log.Warn("failed to get connection: ", err)
			continue
		}
		defer conn.Close()
		log.Info("announce request: ", request)
		_, err = conn.Write(request)
		if err != nil {
			log.Warn("failed to write to UDP tracker: ", err)
			continue
		}

		err = conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			log.Warn("failed to set read deadline: ", err)
			continue
		}

		// Making a long response for getting all the peers
		response := make([]byte, 1024*100)

		numBytesInRes, _, err := conn.ReadFromUDP(response)
		if err != nil {
			log.Warn("failed to read from UDP tracker: ", err)
			continue
		} else if numBytesInRes < 20 {
			log.Warn("received less than 20 bytes: ", numBytesInRes)
		} else {
			// log.Info("announce response: ", response)
			err := c.verifyAnnounceResponsePacket(request, response)
			if err != nil {
				log.Warn("failed to verify announce response packet: ", err)
				continue
			}
			numPeers := (numBytesInRes - 20) / 6
			responseData := c.announceResponsePacketToStruct(response, numPeers)
			log.Info("announce response data: ", responseData)
			return nil
		}
	}

	return nil
}

func NewUDPTrackerClient(host string) TrackerClient {
	return &UDPTrackerClient{host, MAGIC_NUMBER}
}
