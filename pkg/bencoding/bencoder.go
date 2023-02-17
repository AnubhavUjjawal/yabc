package bencoding

import (
	"errors"
	"strconv"
)

type DataType string

const (
	STRING DataType = "string"
	INT    DataType = "int"
	LIST   DataType = "list"
	DICT   DataType = "dict"
)

type Bencoder interface {
	Decode(data string) (interface{}, error)
}

type BencoderImpl struct {
}

func (b *BencoderImpl) isDataString(data string) bool {
	return data[0] >= '0' && data[0] <= '9'
}

func (b *BencoderImpl) isDataList(data string) bool {
	return data[0] == 'l' && data[1] != ':'
}

func (b *BencoderImpl) isDataInt(data string) bool {
	return data[0] == 'i' && data[1] != ':'
}

func (b *BencoderImpl) isDataDict(data string) bool {
	return data[0] == 'd' && data[1] != ':'
}

func (b *BencoderImpl) determineType(data string) (DataType, error) {
	if b.isDataInt(data) {
		return INT, nil
	} else if b.isDataString(data) {
		return STRING, nil
	} else if b.isDataList(data) {
		return LIST, nil
	} else if b.isDataDict(data) {
		return DICT, nil
	}
	return "", errors.New("invalid data format")
}

func (b *BencoderImpl) decodeString(data string) (string, error) {
	if !b.isDataString(data) {
		return "", errors.New("invalid string format")
	}
	return data[2:], nil
}

func (b *BencoderImpl) decodeInt(data string) (int, error) {
	if !b.isDataInt(data) {
		return 0, errors.New("invalid int format")
	}
	return strconv.Atoi(data[1 : len(data)-1])
}

func (b *BencoderImpl) decodeList(data string) ([]interface{}, error) {
	if !b.isDataList(data) {
		return nil, errors.New("invalid list format")
	}
	listItems := make([]interface{}, 0)
	// check the type of the elements in the list as we go.
	// if the type is a complex datatype (list or dict), we need to recursively decode it.
	// if the type is a simple datatype, we can just decode it.

	// we use the two pointer approach to decode the list.
	slow := 1
	fast := 1
	// fmt.Println("hehehhehe")
	for slow < len(data)-2 && fast < len(data)-2 {
		dataType, err := b.determineType(data[slow:])
		if err != nil {
			return nil, err
		}
		if dataType == STRING {
			// move fast pointer to the end of the string
			length := 0
			for data[fast] != ':' {
				length = length*10 + int(data[fast]-'0')
				fast++
			}
			fast = slow + 1 + length
			decodedString, err := b.decodeString(data[slow : fast+1])
			if err != nil {
				return nil, err
			}
			listItems = append(listItems, decodedString)
			slow = fast + 1
			fast = slow
		} else if dataType == INT {
			// move fast pointer to the end of the int
			for data[fast] != 'e' {
				fast++
			}
			decodedInt, err := b.decodeInt(data[slow : fast+1])
			if err != nil {
				return nil, err
			}
			listItems = append(listItems, decodedInt)
			slow = fast + 1
			fast = slow
		}
	}

	return listItems, nil
}

func (b *BencoderImpl) Decode(data string) (interface{}, error) {
	if b.isDataInt(data) {
		return b.decodeInt(data)
	} else if b.isDataString(data) {
		return b.decodeString(data)
	} else if b.isDataList(data) {
		return b.decodeList(data)
	}
	return nil, nil
}

func NewBencoder() Bencoder {
	return &BencoderImpl{}
}
