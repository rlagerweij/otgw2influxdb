package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const cOTGWmsgLength = 9

const (
	cTypeNone  = 0
	cTypeU8    = 1 // unsigned 8-bit integer 0 .. 255
	cTypeU8WDT = 2 // byte representing Day of Week & Time of Day / HB : bits 7,6,5 : day of week / bits 4,3,2,1,0 : hours
	cTypeS8    = 3 // signed 8-bit integer -128 .. 127 (two’s compliment)
	cTypeF8_8  = 4 // signed fixed point value : 1 sign bit, 7 integer bit, 8 fractional bits (two’s compliment ie. the LSB of the 16bit binary number represents 1/256 flag8 byte composed of 8 single-bit flags
	cTypeU16   = 5 // unsigned 16-bit integer 0..65535
	cTypeS16   = 6 // signed 16-bit integer -32768..32767
	cTypeFlag8 = 8 // byte composed of 8 single-bit flags
)

const (
	cReadData      = 0
	cWriteData     = 1
	cInvalidData   = 2
	cReserved      = 3
	cReadAck       = 4
	cWriteAck      = 5
	cDataInvalid   = 6
	cUnknownDataID = 7
)

type openthermMessage struct {
	valid   bool
	msgID   uint8
	msgType uint8
	payload []byte
}

func (ot *openthermMessage) ParseMessage(in string) bool {
	ot.valid = false
	if ot.isValidMsg(in) {
		v, err := hex.DecodeString(in[1:9])
		if err != nil {
			logVerbose.Printf("Message type hex decoder error: %v\n", err.Error())
		} else {
			ot.msgType = uint8((v[0] >> 4) & 7)
			ot.msgID = v[1]
			ot.payload = v[2:]
			ot.valid = true
		}
	}
	return ot.valid
}

func (ot *openthermMessage) DecodeToLineProtocol() string {
	var output, sep string = "", ""

	if ot.valid && ot.isDecodableMsgType() {

		values := ot.decodeValues()

		for n, field := range openthermFieldNames[ot.msgID] {
			if strings.Contains(config[fmt.Sprintf("store_%s", field)], "YES") {
				output += fmt.Sprintf("%s%s=%s", sep, field, values[n])
				sep = "," // prepare for a possible next field
			}
		}

		if len(output) > 0 {
			output = fmt.Sprintf("%s %s %v\n", config["influxMeasurementName"], output, time.Now().Unix())
		}
	}
	return output
}

func (ot *openthermMessage) DecodeToReadable() string {
	var output, sep string = "", ""

	if ot.valid && ot.isDecodableMsgType() {

		values := ot.decodeValues()

		for n, field := range openthermFieldNames[ot.msgID] {
			if strings.Contains(config[fmt.Sprintf("store_%s", field)], "YES") {
				output += fmt.Sprintf("%s%s: %s", sep, openthermReadableNames[ot.msgID][n], values[n])
				sep = "\n" // prepare for a possible next field
			}
		}

	}
	return output
}

func (ot *openthermMessage) decodeValues() []string {
	var output []string

	types := openthermFieldTypes[ot.msgID]

	for index, valueType := range types {
		switch valueType {
		case cTypeFlag8:
			for i := 0; i <= 7; i++ {
				output = append(output, ot.decodeFlag8(index, byte(i)))
			}
		case cTypeF8_8:
			output = append(output, ot.decodeF8_8(ot.payload))
		case cTypeU16:
			output = append(output, ot.decodeU16())
		case cTypeS16:
			output = append(output, ot.decodeS16())
		case cTypeU8:
			output = append(output, ot.decodeU8(index))
		case cTypeS8:
			output = append(output, ot.decodeS8(index))
		case cTypeU8WDT:
			output = append(output, fmt.Sprintf("%v", ot.payload[index]>>5)) // top 3 bits
			output = append(output, fmt.Sprintf("%v", ot.payload[index]&31)) // bottom 5 bits
		case cTypeNone:
		default:
			logVerbose.Println("Unknown opentherm type:", valueType)
		}
	}
	return output
}

func (ot *openthermMessage) bytesToUInt(in []byte) uint16 {
	var result uint16 = 0
	for _, v := range in {
		result <<= 8
		result += uint16(v)
	}
	return result
}

func (ot *openthermMessage) decodeF8_8(in []byte) string {
	// fmt.Println("decoding ", in)
	return fmt.Sprintf("%.2f", ot.bytesToFloat(in))
}

func (ot *openthermMessage) bytesToFloat(in []byte) float64 {
	// fmt.Println("decoding ", in)
	return float64(in[0]) + float64(in[1])/256
}

func (ot *openthermMessage) byteToBool(in byte, bitPosition byte) bool {
	// fmt.Printf("flags % 08b \n", in)
	// fmt.Printf("mask  % 08b %d\n", (1 << bitPosition), bitPosition)
	isFlagSet := (in&(1<<bitPosition) > 0)
	return isFlagSet
}

func (ot *openthermMessage) decodeFlag8(n int, bitPosition byte) string {
	var result = "0"

	if ot.byteToBool(ot.payload[n], bitPosition) {
		result = "1"
	}

	return result
}

func (ot *openthermMessage) decodeU8(n int) string {
	return fmt.Sprintf("%v", ot.bytesToUInt(ot.payload[n:n+1]))
}

func (ot *openthermMessage) decodeS8(n int) string {
	return fmt.Sprintf("%v", int8(ot.bytesToUInt(ot.payload[n:n+1])))
}

func (ot *openthermMessage) decodeU16() string {
	return fmt.Sprintf("%v", ot.bytesToUInt(ot.payload))
}

func (ot *openthermMessage) decodeS16() string {
	return fmt.Sprintf("%v", int16(ot.bytesToUInt(ot.payload)))
}

func (ot *openthermMessage) isValidMsg(msg string) bool {
	var valid = true

	valid = valid && (len(strings.TrimSpace(msg)) == cOTGWmsgLength)
	valid = valid && (msg[0:1] == "T" || msg[0:1] == "B")

	if !valid {
		logVerbose.Print("Received invalid message:", msg)
	}

	return valid
}

func (ot *openthermMessage) isDecodableMsgType() bool {
	if ot.msgType == cDataInvalid ||
		ot.msgType == cUnknownDataID ||
		ot.msgType == cInvalidData {
		logVerbose.Println("OT message contains invalid or unkonw data type:", ot.msgType)
	}
	// only the acknowledgements are worth decoding
	return (ot.msgType == cReadAck || ot.msgType == cWriteAck)
}

var openthermFieldNames = map[uint8][]string{
	0:   {"ch_enabled", "dhw_enabled", "cooling_enabled", "otc_active", "ch2_enabled", "reserved1", "reserved2", "reserved3", "fault_indication", "ch_active", "dhw_active", "flame_active", "cooling_active", "ch2_active", "diagnostic_event", "reserved4"},
	1:   {"control_setpoint"},
	2:   {"master_configuration"},
	3:   {"dhw_present", "control_type", "cooling_supported", "dhw_storage_tank_present", "master_control_allowed", "ch2_present", "reserved", "reserved", "slave_memberID"},
	5:   {"service_required", "remote_reset_enabled", "low_water_pressure_fault", "gas_flame_fault", "air_pressure_fault", "water_over_temperture_fault", "reserved", "reserved", "oem_fault_code"},
	7:   {"cooling_control_signal"},
	8:   {"control_setpoint_2"},
	9:   {"remote_override_room_setpoint"},
	10:  {"number_of_tsps"},
	11:  {"tsp_index", "tsp_value"},
	12:  {"size_of_fault_buffer "},
	13:  {"fhb_fault_index", "fhb_fault_value"},
	14:  {"maximum_relative_modulation_level_setting"},
	15:  {"maximum_boiler_capacity", "minimum_boiler_modulation"},
	16:  {"room_setpoint"},
	17:  {"relative_modulation_level"},
	18:  {"ch_water_pressure"},
	19:  {"dhw_flow_rate"},
	20:  {"weekday", "hour", "minutes"},
	21:  {"month", "day"},
	22:  {"year"},
	23:  {"room_setpoint_ch2"},
	24:  {"room_temperature"},
	25:  {"boiler_water_temp"},
	26:  {"dhw_temperature"},
	27:  {"outside_temperature"},
	28:  {"return_water_temperature"},
	29:  {"solar_storage_temperature"},
	30:  {"solar_collector_temperature"},
	31:  {"flow_temperature_ch2"},
	32:  {"dhw2_temperature"},
	33:  {"exhaust_temperature"},
	48:  {"dhwsetpoint_upper_bound", "dhwsetpoint_lower_bound"},
	49:  {"max_chsetp_upper_bound", "max_chsetp_lower_bound"},
	56:  {"dhw_setpoint"},
	57:  {"max_ch_water_setpoint"},
	100: {"manual_setpoint_overrules_remote_setpoint", "program_change_setpoint_overrides_remote_setpoint", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"},
	115: {"oem_diagnostic_code"},
	116: {"burner_starts"},
	117: {"ch_pump_starts"},
	118: {"dhw_pump_valve_starts"},
	119: {"dhw_burner_starts"},
	120: {"burner_operation_hours"},
	121: {"ch_pump_operation_hours"},
	122: {"dhw_pump_valve_operation_hours"},
	123: {"dhw_burner_operation_hours"},
	124: {"opentherm_version_master"},
	125: {"opentherm_version_slave"},
	126: {"master_product_version_number", "master_product_type"},
	127: {"slave_product_version_number", "slave_product_type"},
}
var openthermReadableNames = map[uint8][]string{
	0:   {"CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved"},
	1:   {"Temperature setpoint for the supply from the boiler (°C)"},
	2:   {"MemberID code of the master"},
	3:   {"DHW present [ dhw not present, dhw is present ]", "Control type [ modulating, on/off ]", "Cooling config [ cooling not supported, cooling supported]", "DHW config [instantaneous or not-specified, storage tank]", "Master low-off&pump control function [allowed, not allowed]", "CH2 present [CH2 not present, CH2 present]", "reserved", "reserved", "MemberID code of the slave"},
	5:   {"Service request [service not req’d, service required]", "Lockout-reset [ remote reset disabled, rr enabled]", "Low water press [no WP fault, water pressure fault]", "Gas/flame fault [ no G/F fault, gas/flame fault ]", "Air press fault [ no AP fault, air pressure fault ]", "Water over-temp[no OvT fault, over-temperat. Fault]", "reserved", "reserved", "OEM fault code u8 0..255 An OEM-specific fault/error code"},
	7:   {"Signal for cooling plant"},
	8:   {"Temperature setpoint for the supply from the boiler for circuit 2 (°C)"},
	9:   {"Remote override room setpoint (0 = No override)"},
	10:  {"Number of transparent-slave-parameter supported by the slave device"},
	11:  {"Index number of following TSP", "Value of above referenced TSP"},
	12:  {"The size of the fault history buffer"},
	13:  {"Index number of following Fault Buffer entry", "Value of above referenced Fault Buffer entry"},
	14:  {"Maximum relative boiler modulation level setting for sequencer and off-low&pump control applications (%)"},
	15:  {"Maximum boiler capacity (kW)", "Minimum modulation level (%)"},
	16:  {"Current room temperature setpoint (°C)"},
	17:  {"Relative modulation level (%)"},
	18:  {"Water pressure of the boiler CH circuit (bar)"},
	19:  {"Water flow rate through the DHW circuit (l/min)"},
	20:  {"Day of the week (1=Monday)", "Hours", "Minutes"},
	21:  {"Month", "Day of Month"},
	22:  {"Year"},
	23:  {"Current room setpoint for 2nd CH circuit (°C)"},
	24:  {"Current sensed room temperature (°C)"},
	25:  {"Flow water temperature from boiler (°C)"},
	26:  {"Domestic hot water temperature (°C)"},
	27:  {"Outside air temperature (°C)"},
	28:  {"Return water temperature to boiler (°C)"},
	29:  {"Solar storage temperature (°C)"},
	30:  {"Solar collector temperature (°C)"},
	31:  {"Flow water temperature of the second central heating circuit"},
	32:  {"Domestic hot water temperature 2 (°C)"},
	33:  {"Exhaust temperature (°C)"},
	48:  {"Upper bound for adjustment of DHW setp (°C)", "Lower bound for adjustment of DHW setp (°C)"},
	49:  {"Upper bound for adjustment of maxCHsetp (°C)", "Lower bound for adjustment of maxCHsetp (°C)"},
	56:  {"Domestic hot water temperature setpoint (°C)"},
	57:  {"Maximum allowable CH water setpoint (°C)"},
	100: {"Manual change priority [0 = disable overruling remote setpoint by manual setpoint change, 1 = enable overruling remote setpoint by manual setpoint change]", "Program change priority [0 = disable overruling remote setpoint by program setpoint change, 1 = enable overruling remote setpoint by program setpoint change]", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"},
	115: {"OEM-specific diagnostic/service code"},
	116: {"Number of starts burner"},
	117: {"Number of starts CH pump"},
	118: {"Number of starts DHW pump/valve"},
	119: {"Number of starts burner in DHW mode"},
	120: {"Number of hours that burner is in operation (i.e.flame on)"},
	121: {"Number of hours that CH pump has been running"},
	122: {"Number of hours that DHW pump has been running or DHW valve has been opened"},
	123: {"Number of hours that burner is in operation during DHW mode"},
	124: {"The implemented version of the OpenTherm Protocol Specification in the master"},
	125: {"The implemented version of the OpenTherm Protocol Specification in the slave"},
	126: {"The master device product version number as defined by the manufacturer", "The master device product type as defined by the manufacturer"},
	127: {"The slave device product version number as defined by the manufacturer", "The slave device product type as defined by the manufacturer"},
}

var openthermFieldTypes = map[uint8][]uint8{
	0:   {cTypeFlag8, cTypeFlag8},
	1:   {cTypeF8_8, cTypeNone},
	2:   {cTypeNone, cTypeU8},
	3:   {cTypeFlag8, cTypeU8},
	5:   {cTypeFlag8, cTypeU8},
	7:   {cTypeF8_8, cTypeNone},
	8:   {cTypeF8_8, cTypeNone},
	9:   {cTypeF8_8, cTypeNone},
	10:  {cTypeU8, cTypeU8},
	11:  {cTypeU8, cTypeU8},
	12:  {cTypeU8, cTypeNone},
	13:  {cTypeU8, cTypeU8},
	14:  {cTypeF8_8, cTypeNone},
	15:  {cTypeU8, cTypeU8},
	16:  {cTypeF8_8, cTypeNone},
	17:  {cTypeF8_8, cTypeNone},
	18:  {cTypeF8_8, cTypeNone},
	19:  {cTypeF8_8, cTypeNone},
	20:  {cTypeU8WDT, cTypeU8},
	21:  {cTypeU8, cTypeU8},
	22:  {cTypeU16, cTypeNone},
	23:  {cTypeF8_8, cTypeNone},
	24:  {cTypeF8_8, cTypeNone},
	25:  {cTypeF8_8, cTypeNone},
	26:  {cTypeF8_8, cTypeNone},
	27:  {cTypeF8_8, cTypeNone},
	28:  {cTypeF8_8, cTypeNone},
	29:  {cTypeF8_8, cTypeNone},
	30:  {cTypeS16, cTypeNone},
	31:  {cTypeF8_8, cTypeNone},
	32:  {cTypeF8_8, cTypeNone},
	33:  {cTypeS16, cTypeNone},
	48:  {cTypeS8, cTypeS8},
	49:  {cTypeS8, cTypeS8},
	56:  {cTypeF8_8, cTypeNone},
	57:  {cTypeF8_8, cTypeNone},
	100: {cTypeNone, cTypeFlag8},
	115: {cTypeU16, cTypeNone},
	116: {cTypeU16, cTypeNone},
	117: {cTypeU16, cTypeNone},
	118: {cTypeU16, cTypeNone},
	119: {cTypeU16, cTypeNone},
	120: {cTypeU16, cTypeNone},
	121: {cTypeU16, cTypeNone},
	122: {cTypeU16, cTypeNone},
	123: {cTypeU16, cTypeNone},
	124: {cTypeF8_8, cTypeNone},
	125: {cTypeF8_8, cTypeNone},
	126: {cTypeU8, cTypeU8},
	127: {cTypeU8, cTypeU8},
}
