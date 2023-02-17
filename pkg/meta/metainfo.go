package meta

type FileData struct {
	Length int      `json:"length" bencode:"length"`
	Path   []string `json:"path" bencode:"path"`
}

type InfoDict struct {
	Files       []FileData `json:"files" bencode:"files"`
	Name        string     `json:"name" bencode:"name"`
	PieceLength int        `json:"piece length" bencode:"piece length"`
	Pieces      string     `json:"pieces" bencode:"pieces"`
}

type MetaInfo struct {
	Announce     string     `json:"announce" bencode:"announce"`
	AnnounceList [][]string `json:"announce-list" bencode:"announce-list"`
	Comment      string     `json:"comment" bencode:"comment"`
	CreatedBy    string     `json:"created by" bencode:"created by"`
	CreationDate int        `json:"creation date" bencode:"creation date"`
	Info         InfoDict   `json:"info" bencode:"info"`
}
