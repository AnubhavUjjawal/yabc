package meta

import (
	"fmt"
)

type FileData struct {
	Length int      `json:"length" bencode:"length"`
	Path   []string `json:"path" bencode:"path"`
}

// TODO: Add support for single file torrents
type InfoDict struct {
	Files       []FileData `json:"files" bencode:"files"`
	Name        string     `json:"name" bencode:"name"`
	Length      int        `json:"length" bencode:"length"`
	PieceLength int        `json:"piece length" bencode:"piece length"`
	Pieces      string     `json:"pieces" bencode:"pieces"`
}

type BlockRequest struct {
	Index  int
	Begin  int
	Length int
}

type BlockResponse struct {
	BlockRequest
	Data []byte
}

// Pieces is a string of 20 byte SHA1 hashes concatenated together.
// We read each byte and convert it to it's hex representation, along with padding if needed.
// This is the actual SHA1 hash concatenated together.

// https://stackoverflow.com/a/54648809/13499618
// https://wiki.theory.org/BitTorrentSpecification (Info Dictionary Section)
func (idict InfoDict) GetPiecesHashes() []string {
	shaHashString := ""
	hashes := make([]string, 0)
	for i := 0; i < len(idict.Pieces); i += 1 {
		shaHashString += fmt.Sprintf("%02x", byte(idict.Pieces[i]))
	}
	for i := 0; i < len(shaHashString); i += 40 {
		hashes = append(hashes, shaHashString[i:i+40])
	}
	return hashes
}

type MetaInfo struct {
	Announce     string     `json:"announce" bencode:"announce"`
	AnnounceList [][]string `json:"announce-list" bencode:"announce-list"`
	Comment      string     `json:"comment" bencode:"comment"`
	CreatedBy    string     `json:"created by" bencode:"created by"`
	CreationDate int        `json:"creation date" bencode:"creation date"`
	Info         InfoDict   `json:"info" bencode:"info"`

	// RawData is the raw string of the torrent file.
	// We need it to calculate the infohash of the torrent.
	RawData string `json:"raw_data" bencode:"raw_data"`
}
