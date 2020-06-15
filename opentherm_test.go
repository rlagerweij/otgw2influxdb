package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeF8_8(t *testing.T) {
	testTable := []struct {
		in  string
		out string
	}{
		{"00FF", "1.00"},
		{"0000", "0.00"},
		{"13A0", "19.62"},
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

func TestDecodeFlag8(t *testing.T) {
	testTable := []struct {
		in     string
		bitNum byte
		out    string
	}{
		{"FFFF", 1, "1"},
		{"0000", 1, "0"},
	}
	for _, test := range testTable {
		arg, err := hex.DecodeString(test.in)
		if err != nil {
			t.Errorf("decodig hex string %s failed: %v", test.in, err)
		} else {
			actual := decodeFlag8(arg[0], test.bitNum)
			if actual != test.out {
				t.Errorf("decodeFlag(%v) failed: expected %s, got %s", arg, test.out, actual)
			}
		}
	}
}

func TestDecodeLP(t *testing.T) {
	var passed bool = true
	// var testedFunction = *func()

	testTable := []struct {
		in  string
		out string
	}{
		{"B40193C33", "otgw boiler_water_temp=60.20 "},      // cTypeF8_8
		{"BC0784750", "otgw burner_operation_hours=18256 "}, //cTypeU16
	}
	readConfig("otgw2db.testing.cfg")

	for _, test := range testTable {
		result := decodeLineProtocol(test.in)
		parts := strings.SplitAfter(result, " ")
		actual := fmt.Sprint(parts[0], parts[1])

		if strings.Compare(actual, test.out) != 0 {
			passed = false
		}
		if !passed {
			t.Errorf("decodeLP(\"%v\") failed: expected %s, got %s", test.in, test.out, actual)
		}
	}
}

func TestConfigWithDecodeLP(t *testing.T) {

	testTable := []struct {
		in         string
		settingKey string
		out        string
		out2       string
	}{
		{"B40193C33", "store_boiler_water_temp", "otgw boiler_water_temp=60.20", ""},
	}

	readConfig("otgw2db.testing.cfg")

	for _, test := range testTable {
		config[test.settingKey] = "YES"
		actual := decodeLineProtocol(test.in)
		if !strings.Contains(actual, test.out) {
			t.Errorf("decodeLP(\"%v\") with setting %s failed: expected %s, got %s", test.in, config[test.settingKey], test.out, actual)
		}
		config[test.settingKey] = "NO"
		actual2 := decodeLineProtocol(test.in)
		if !strings.Contains(actual2, test.out2) {
			t.Errorf("decodeLP(\"%v\") with setting %s failed: expected %s, got %s", test.in, config[test.settingKey], test.out2, actual2)
		}
	}
}
