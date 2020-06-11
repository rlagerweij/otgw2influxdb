package main

import (
	"encoding/hex"
	"testing"
)

func TestDecodeF8_8(t *testing.T) {
	testTable := []struct {
		in  string
		out string
	}{
		{"00FF", "1.00"},
		{"0000", "0.00"},
	}
	for _, test := range testTable {
		arg, err := hex.DecodeString(test.in)
		if err != nil {
			t.Errorf("decodig hex string %s failed: %v", test.in, err)
		} else {
			actual := decodeF8_8(arg)
			if actual != test.out {
				t.Errorf("decodeF8_8(%v) failed: expected %s, got %s", arg, test.out, actual)
			}
		}
	}
}
