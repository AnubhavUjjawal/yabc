package bencoding_test

import (
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
			decoded, err := a.Decode(test[0])
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
			decoded, err := a.Decode(test[0].(string))
			if err != nil {
				t.Errorf("Error decoding: %v", err)
			}
			if decoded != test[1] {
				t.Errorf("Expected: %v, got: %v", test[1], decoded)
			}
		})
	}
}

// func TestBencoderListDecodeIsWorking(t *testing.T) {
// 	a := bencoding.NewBencoder()
// 	tests := [][]interface{}{
// 		{"li5ei7e5:helloe", []interface{}{5, 7, "hello"}},
// 		{"li-5ei7e5:helloe", []interface{}{-5, 7, "hello"}},
// 		{"l4:spam4:eggse", []interface{}{"spam", "eggs"}},
// 		{"le", []interface{}{}},
// 		{"l5:helloi5ei7ee", []interface{}{"hello", 5, 7}},
// 		{"l4:spami4ee", []interface{}{"spam", 4}},
// 		{"l4:spami4el5:hello5:worlde3:anti3ee", []interface{}{"spam", 4, []interface{}{"hello", "world"}, "ant", 3}},
// 	}
// 	for _, test := range tests {
// 		t.Run(test[0].(string), func(t *testing.T) {
// 			decoded, err := a.Decode(test[0].(string))
// 			if err != nil {
// 				t.Errorf("Error decoding: %v", err)
// 			}
// 			// log.Println(decoded)
// 			if !reflect.DeepEqual(decoded, test[1]) {
// 				t.Errorf("Expected: %v, got: %v", test[1], decoded)
// 			}
// 		})
// 	}
// }

// func TestBencoderDictDecodeIsWorking(t *testing.T) {
// 	a := bencoding.NewBencoder()
// 	tests := [][]interface{}{
// 		{"d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": "moo", "spam": "eggs"}},
// 		{"d4:spaml1:a1:bee", map[string]interface{}{"spam": []string{"a", "b"}}},
// 	}
// 	for _, test := range tests {
// 		t.Run(test[0].(string), func(t *testing.T) {
// 			decoded, err := a.Decode(test[0].(string))
// 			if err != nil {
// 				t.Errorf("Error decoding: %v", err)
// 			}
// 			// log.Println(decoded)
// 			if !reflect.DeepEqual(decoded, test[1]) {
// 				t.Errorf("Expected: %v, got: %v", test[1], decoded)
// 			}
// 		})
// 	}
// }
