package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfigNew(t *testing.T) {

	testTable := []struct {
		in         string
		settingKey string
		out        string
		out2       string
	}{
		{"B40193C33", "store_boiler_water_temp", "otgw boiler_water_temp=60.20", ""},
	}

	testOT := openthermMessage{}

	readConfig("otgw2db.testing.cfg")

	for _, test := range testTable {
		config[test.settingKey] = "YES"
		testOT.ParseMessage(test.in)
		actual := testOT.DecodeToLineProtocol()
		if !strings.Contains(actual, test.out) {
			t.Errorf("decodeLP(\"%v\") with setting %s failed: expected %s, got %s", test.in, config[test.settingKey], test.out, actual)
		}
		config[test.settingKey] = "NO"
		actual2 := testOT.DecodeToLineProtocol()
		if !strings.Contains(actual2, test.out2) {
			t.Errorf("decodeLP(\"%v\") with setting %s failed: expected %s, got %s", test.in, config[test.settingKey], test.out2, actual2)
		}
	}
}

func TestNewDecodeLP(t *testing.T) {

	testTable := []struct {
		in  string
		out string
	}{
		{"B40193C33", "otgw boiler_water_temp=60.20 "},      // cTypeF8_8
		{"BC0784750", "otgw burner_operation_hours=18256 "}, //cTypeU16
		{"B40000200", "otgw ch_enabled=0,dhw_enabled=1,cooling_enabled=0,otc_active=0,ch2_enabled=0,fault_indication=0,ch_active=0,dhw_active=0,flame_active=0,cooling_active=0,ch2_active=0,diagnostic_event=0"}, //cTypeFlag8
		{"B407F0511", "otgw slave_product_version_number=5,slave_product_type=17"},  //cTypeU8
		{"BC0303C28", "otgw dhwsetpoint_upper_bound=60,dhwsetpoint_lower_bound=40"}, //cTypeS8
	}
	readConfig("otgw2db.testing.cfg")
	testOT := openthermMessage{}

	for _, test := range testTable {
		_ = testOT.ParseMessage(test.in)
		result := testOT.DecodeToLineProtocol()
		if len(result) > 10 {
			if !strings.Contains(result, test.out) {
				t.Errorf("NewDecodeLP(\"%v\") failed: expected \"%s\", got \"%s\"", test.in, test.out, result)
			}
		}
	}
}

func TestNewDecodeReadable(t *testing.T) {

	testTable := []struct {
		in  string
		out string
	}{
		{"B40191F80", "Flow water temperature from boiler (°C): 31.50"}, // cTypeF8_8
	}
	readConfig("otgw2db.testing.cfg")
	testOT := openthermMessage{}

	for _, test := range testTable {
		_ = testOT.ParseMessage(test.in)
		result := testOT.DecodeToReadable()
		fmt.Println(result)
		if len(result) > 10 {
			if !strings.Contains(result, test.out) {
				t.Errorf("NewDecodeReadable(\"%v\") failed: expected \"%s\", got \"%s\"", test.in, test.out, result)
			}
		}
	}
}
