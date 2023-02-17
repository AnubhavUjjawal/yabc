package bencoding_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/AnubhavUjjawal/yabc/pkg/bencoding"
)

func TestBencoderStringDecodeIsWorking(t *testing.T) {
	a := bencoding.NewBencoder()
	tests := [][]string{
		{"5:hello", "hello"},
		{"7:anubhav", "anubhav"},
		{"5:world", "world"},
	}
	for _, test := range tests {
		t.Run(test[0], func(t *testing.T) {
			decoded, _, err := a.Decode(test[0])
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			if decoded != test[1] {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

func TestBencoderIntDecodeIsWorking(t *testing.T) {
	a := bencoding.NewBencoder()
	tests := [][]interface{}{
		{"i5e", 5},
		{"i7e", 7},
		{"i-5e", -5},
	}
	for _, test := range tests {
		t.Run(test[0].(string), func(t *testing.T) {
			decoded, _, err := a.Decode(test[0].(string))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			if decoded != test[1] {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

func TestBencoderListDecodeIsWorking(t *testing.T) {
	a := bencoding.NewBencoder()
	tests := [][]interface{}{
		{"li5ei7e5:helloe", []interface{}{5, 7, "hello"}},
		{"li-5ei7e5:helloe", []interface{}{-5, 7, "hello"}},
		{"l4:spam4:eggse", []interface{}{"spam", "eggs"}},
		{"le", []interface{}{}},
		{"l5:helloi5ei7ee", []interface{}{"hello", 5, 7}},
		{"l4:spami4ee", []interface{}{"spam", 4}},
		{"l4:spami4el5:hello5:worlde3:anti3ee", []interface{}{"spam", 4, []interface{}{"hello", "world"}, "ant", 3}},
		{"l4:spami4el5:hello5:worldl4:abcde3:anti3eee", []interface{}{"spam", 4, []interface{}{"hello", "world", []interface{}{"abcd"}, "ant", 3}}},
	}
	for _, test := range tests {
		t.Run(test[0].(string), func(t *testing.T) {
			decoded, _, err := a.Decode(test[0].(string))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			// log.Println(decoded)
			if !reflect.DeepEqual(decoded, test[1]) {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

func TestBencoderDictDecodeIsWorking(t *testing.T) {
	a := bencoding.NewBencoder()
	tests := [][]interface{}{
		{"d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": "moo", "spam": "eggs"}},
		{"d3:cow3:moo4:spami34ee", map[string]interface{}{"cow": "moo", "spam": 34}},
		{"d4:spaml1:a1:bee", map[string]interface{}{"spam": []interface{}{"a", "b"}}},
	}
	for _, test := range tests {
		t.Run(test[0].(string), func(t *testing.T) {
			decoded, _, err := a.Decode(test[0].(string))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			// log.Println(decoded)
			if !reflect.DeepEqual(decoded, test[1]) {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

func TestNestedDict(t *testing.T) {
	a := bencoding.NewBencoder()
	tests := [][]interface{}{
		{
			"d3:cowd5:sound3:mooee",
			map[string]interface{}{
				"cow": map[string]interface{}{
					"sound": "moo",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test[0].(string), func(t *testing.T) {
			decoded, _, err := a.Decode(test[0].(string))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			// log.Println(decoded)
			if !reflect.DeepEqual(decoded, test[1]) {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

func TestComplexString(t *testing.T) {
	a := bencoding.NewBencoder()
	str := "d8:announce42:udp://tracker.opentrackr.org:1337/announce13:announce-listll42:udp://tracker.opentrackr.org:1337/announceel49:udp://tracker.leechers-paradise.org:6969/announceel30:udp://9.rarbg.to:2710/announceel34:udp://p4p.arenabg.ch:1337/announceel38:udp://tracker.cyberia.is:6969/announceel36:http://p4p.arenabg.com:1337/announceel48:udp://tracker.internetwarriors.net:1337/announceee7:comment70:Torrent downloaded from https://yts.mx ( former yts.Lt/yts.am/yts.ag )10:created by64:Torrent RW PHP Class - http://github.com/adriengibrat/torrent-rw13:creation datei1628269273e4:infod5:filesld6:lengthi1413009249e4:pathl62:Mulholland.Drive.2001.REPACK.720p.BluRay.x264.AAC-[YTS.MX].mp4eed6:lengthi358e4:pathl18:YIFYStatus.com.txteed6:lengthi53226e4:pathl14:www.YTS.MX.jpgeee4:name57:Mulholland Drive (2001) [REPACK] [720p] [BluRay] [YTS.MX]12:piece lengthi1380352eee"

	decoded, _, err := a.Decode(str)
	if err != nil {
		t.Errorf("Error decoding: %v", err)
	}
	t.Log(decoded)

}

func TestDecodeIsWorkingOnSampleFiles(t *testing.T) {
	a := bencoding.NewBencoder()
	rootDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting root dir: %v", err)
	}
	files, err := ioutil.ReadDir(filepath.Join(rootDir, "..", "..", "sample_torrents"))
	if err != nil {
		t.Errorf("Error reading sample files: %v", err)
	}
	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(rootDir, "..", "..", "sample_torrents", file.Name()))
			if err != nil {
				t.Errorf("Error opening file: %v", err)
			}
			_, _, err = a.Decode(string(content))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			// t.Log(decoded)
		})
	}
}

func TestUnmarshal(t *testing.T) {
	a := bencoding.NewBencoder()
	type DataStruct struct {
		Files []struct {
			Path []string `bencode:"path"`
		} `bencode:"files"`
	}
	str := "d5:filesld4:pathl6:path-1eed4:pathl6:path-2eeee"
	t.Run("Testing unmarshal", func(t *testing.T) {
		var data DataStruct
		err := a.Unmarshal(str, &data)
		if err != nil {
			t.Errorf("Error unmarshaling: %v", err)
		}
		result := DataStruct{
			Files: []struct {
				Path []string `bencode:"path"`
			}{
				{
					Path: []string{"path-1"},
				},
				{
					Path: []string{"path-2"},
				},
			},
		}
		if !reflect.DeepEqual(data, result) {
			t.Errorf("Expected: %v, got: %v", result, data)
		}
	})
}
