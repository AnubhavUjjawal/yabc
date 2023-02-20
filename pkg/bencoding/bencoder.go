package bencoding

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/mitchellh/mapstructure"
)

type DataType string

const (
	STRING DataType = "string"
	INT    DataType = "int"
	LIST   DataType = "list"
	DICT   DataType = "dict"
)

type Bencoder interface {
	// Decode returns the decoded data, current position of the cursor in the `data string`` and an error, if any.
	Decode(data string) (interface{}, int, error)
	Unmarshal(data string, dt interface{}) error

	// NOTE: this method does not belong here. Refactor this.
	GetRawValueFromDict(data string, key string) (string, error)
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
	// fmt.Println(data, data[0] == 'd' && data[1] != ':')
	return data[0] == 'd' && data[1] != ':'
}

// func (b *BencoderImpl) determineType(data string) (DataType, error) {
// 	if b.isDataInt(data) {
// 		return INT, nil
// 	} else if b.isDataString(data) {
// 		return STRING, nil
// 	} else if b.isDataList(data) {
// 		return LIST, nil
// 	} else if b.isDataDict(data) {
// 		return DICT, nil
// 	}
// 	return "", errors.New("invalid data format")
// }

func (b *BencoderImpl) decodeString(data string) (string, int, error) {
	if !b.isDataString(data) {
		return "", 0, errors.New("invalid string format")
	}
	idx := 0
	length := 0
	for data[idx] != ':' {
		length = length*10 + int(data[idx]-'0')
		idx++
	}
	return data[idx+1 : idx+1+length], idx + 1 + length, nil
}

func (b *BencoderImpl) decodeInt(data string) (int, int, error) {
	if !b.isDataInt(data) {
		return 0, 0, errors.New("invalid int format")
	}
	intEnd := 0
	for data[intEnd] != 'e' {
		intEnd++
	}

	val, err := strconv.Atoi(data[1:intEnd])
	return val, intEnd + 1, err
}

func (b *BencoderImpl) decodeDict(data string) (interface{}, int, error) {
	if !b.isDataDict(data) {
		return nil, 0, errors.New("invalid dict format")
	}
	dict := make(map[string]interface{})
	// the key is always a string
	// the value can be a string, int, list or dict
	slow := 1
	for slow < len(data)-2 && data[slow] != 'e' {
		key, pointer, err := b.Decode(data[slow:])
		if err != nil {
			return nil, 0, err
		}
		slow += pointer

		valString := data[slow:]
		val, pointer, err := b.Decode(valString)
		dict[key.(string)] = val

		if err != nil {
			return nil, 0, err
		}

		slow += pointer
	}

	return dict, slow + 1, nil
}

func (b *BencoderImpl) decodeList(data string) ([]interface{}, int, error) {
	if !b.isDataList(data) {
		return nil, 0, errors.New("invalid list format")
	}
	listItems := make([]interface{}, 0)
	// check the type of the elements in the list as we go.
	// if the type is a complex datatype (list or dict), we need to recursively decode it.
	// if the type is a simple datatype, we can just decode it.

	// we use the two pointer approach to decode the list.
	slow := 1
	for slow < len(data)-2 && data[slow] != 'e' {
		value, pointer, err := b.Decode(data[slow:])
		if err != nil {
			return nil, 0, err
		}
		listItems = append(listItems, value)
		slow += pointer
	}

	return listItems, slow + 1, nil
}

func (b *BencoderImpl) Decode(data string) (interface{}, int, error) {
	if b.isDataInt(data) {
		return b.decodeInt(data)
	} else if b.isDataString(data) {
		return b.decodeString(data)
	} else if b.isDataList(data) {
		return b.decodeList(data)
	} else if b.isDataDict(data) {
		return b.decodeDict(data)
	}
	return nil, 0, nil
}

func (b *BencoderImpl) Unmarshal(data string, dt interface{}) error {
	decoded, _, err := b.Decode(data)
	if err != nil {
		return err
	}
	if reflect.ValueOf(decoded).Kind() == reflect.Map {
		// decoded = map[string]interface{}(decoded.(map[string]interface{}))
		decoded.(map[string]interface{})["raw_data"] = data
	}
	// dt = decoded
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   dt,
		TagName:  "bencode",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err = decoder.Decode(decoded)
	return err
}

func (b *BencoderImpl) GetRawValueFromDict(data string, key string) (string, error) {
	if !b.isDataDict(data) {
		return "", errors.New("invalid dict format")
	}
	// seek through the dict till we find the key
	slow := 1
	for slow < len(data)-2 && data[slow] != 'e' {
		// decode the key
		keyString, pointer, err := b.Decode(data[slow:])
		if err != nil {
			return "", err
		}
		slow += pointer
		if keyString.(string) == key {
			// we found the key, we need to return the raw value
			valString := data[slow:]
			_, pointer, err := b.Decode(valString)
			if err != nil {
				return "", err
			}
			return valString[:pointer], nil
		}
		// we did not find the key
		// skip the value
		valString := data[slow:]
		_, pointer, err = b.Decode(valString)
		if err != nil {
			return "", err
		}
		slow += pointer
	}
	return "", errors.New("key not found")
}

func NewBencoder() Bencoder {
	return &BencoderImpl{}
}
