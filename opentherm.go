package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

const cOTGWmsgLength = 11

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

const (
	cFieldMaskBit1     = 1 << 0
	cFieldMaskBit2     = 1 << 1
	cFieldMaskBit3     = 1 << 1
	cFieldMaskBit4     = 1 << 1
	cFieldMaskBit5     = 1 << 1
	cFieldMaskBit6     = 1 << 1
	cFieldMaskBit7     = 1 << 1
	cFieldMaskBit8     = 1 << 1
	cFieldMaskBit9     = 1 << 1
	cFieldMaskBit10    = 1 << 1
	cFieldMaskBit11    = 1 << 1
	cFieldMaskBit12    = 1 << 1
	cFieldMaskBit13    = 1 << 1
	cFieldMaskBit14    = 1 << 1
	cFieldMaskBit15    = 1 << 1
	cFieldMaskBit16    = 1 << 1
	cFieldMaskLowByte  = 255 << 0
	cFieldMaskHighByte = 255 << 8
	cFieldMaskAllBits  = cFieldMaskHighByte + cFieldMaskLowByte
)

type openthermMessage struct {
	message []byte
}

type oTValue struct {
	fields       []string
	highByteType uint8
	lowByteType  uint8
	descriptions []string
}

type oTInfluxField struct {
	fieldName string
	fieldMask uint16
	fieldType uint8
}

type oTValueInflux struct {
	fields []oTInfluxField
}

var decoderMapInflux = map[uint8]oTValueInflux{
	0: oTValueInflux{[]oTInfluxField{{"CH_status", cFieldMaskBit2, cTypeF8_8}, {"DHW_status", cFieldMaskBit3, cTypeF8_8}, {"Flame_status", cFieldMaskBit4, cTypeF8_8}, {"Cooling_status", cFieldMaskBit5, cTypeF8_8}, {"CH2_status", cFieldMaskBit6, cTypeF8_8}, {"Diagnostic_Event", cFieldMaskBit7, cTypeF8_8}}},
}

var decoderMapReadable = map[uint8]oTValue{
	0:   oTValue{[]string{"ch_enabled", "dhw_enabled", "cooling_enabled", "otc_active", "ch2_enabled", "reserved1", "reserved2", "reserved3", "fault_indication", "ch_active", "dhw_active", "flame_active", "cooling_active", "ch2_active", "diagnostic_event", "reserved4"}, cTypeFlag8, cTypeFlag8, []string{"CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved"}},
	1:   oTValue{[]string{"control_setpoint"}, cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler in degrees C"}},
	2:   oTValue{[]string{"master_configuration"}, cTypeNone, cTypeU8, []string{"MemberID code of the master"}},
	3:   oTValue{[]string{"dhw_present", "control_type", "cooling_supported", "dhw_storage_tank_present", "master_control_allowed", "ch2_present", "reserved", "reserved", "reserved", "slave_memberID"}, cTypeFlag8, cTypeU8, []string{"DHW present [ dhw not present, dhw is present ]", "Control type [ modulating, on/off ]", "Cooling config [ cooling not supported, cooling supported]", "DHW config [instantaneous or not-specified, storage tank]", "Master low-off&pump control function [allowed, not allowed]", "CH2 present [CH2 not present, CH2 present]", "reserved", "reserved", "reserved", "MemberID code of the slave"}},
	5:   oTValue{[]string{"service_required", "remote_reset_enabled", "low_water_pressure_fault", "gas_flame_fault", "air_pressure_fault", "water_over_temperture_fault", "reserved", "reserved", "oem_fault_code"}, cTypeFlag8, cTypeU8, []string{"Service request [service not req’d, service required]", "Lockout-reset [ remote reset disabled, rr enabled]", "Low water press [no WP fault, water pressure fault]", "Gas/flame fault [ no G/F fault, gas/flame fault ]", "Air press fault [ no AP fault, air pressure fault ]", "Water over-temp[no OvT fault, over-temperat. Fault]", "reserved", "reserved", "OEM fault code u8 0..255 An OEM-specific fault/error code"}},
	7:   oTValue{[]string{"cooling_control_signal"}, cTypeF8_8, cTypeNone, []string{"Signal for cooling plant"}},
	8:   oTValue{[]string{"control_setpoint_2"}, cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler for circuit 2 in degrees C"}},
	9:   oTValue{[]string{"remote_override_room_setpoint"}, cTypeF8_8, cTypeNone, []string{"Remote override room setpoint (0 = No override)"}},
	10:  oTValue{[]string{"number_of_tsps"}, cTypeU8, cTypeU8, []string{"Number of transparent-slave-parameter supported by the slave device"}},
	11:  oTValue{[]string{"tsp_index", "tsp_value"}, cTypeU8, cTypeU8, []string{"Index number of following TSP", "Value of above referenced TSP"}},
	12:  oTValue{[]string{"size_of_fault_buffer "}, cTypeU8, cTypeNone, []string{"The size of the fault history buffer"}},
	13:  oTValue{[]string{"fhb_fault_index", "fhb_fault_value"}, cTypeU8, cTypeU8, []string{"Index number of following Fault Buffer entry", "Value of above referenced Fault Buffer entry"}},
	14:  oTValue{[]string{"maximum_relative_modulation_level_setting"}, cTypeF8_8, cTypeNone, []string{"Maximum relative boiler modulation level setting for sequencer and off-low&pump control applications (%)"}},
	15:  oTValue{[]string{"maximum_boiler_capacity", "minimum_boiler_modulation"}, cTypeU8, cTypeU8, []string{"Maximum boiler capacity (kW)", "Minimum modulation level (%)"}},
	16:  oTValue{[]string{"room_setpoint"}, cTypeF8_8, cTypeNone, []string{"Current room temperature setpoint (°C)"}},
	17:  oTValue{[]string{"relative_modulation_level"}, cTypeF8_8, cTypeNone, []string{"Relative modulation level (%)"}},
	18:  oTValue{[]string{"ch_water_pressure"}, cTypeF8_8, cTypeNone, []string{"Water pressure of the boiler CH circuit (bar)"}},
	19:  oTValue{[]string{"dhw_flow_rate"}, cTypeF8_8, cTypeNone, []string{"Water flow rate through the DHW circuit (l/min)"}},
	20:  oTValue{[]string{"weekday", "hour", "minutes"}, cTypeU8WDT, cTypeU8, []string{"Day of the week (1=Monday)", "Hours", "Minutes"}},
	21:  oTValue{[]string{"month", "day"}, cTypeU8, cTypeU8, []string{"Month", "Day of Month"}},
	22:  oTValue{[]string{"year"}, cTypeU16, cTypeNone, []string{"Year"}},
	23:  oTValue{[]string{"room_setpoint_ch2"}, cTypeF8_8, cTypeNone, []string{"Current room setpoint for 2nd CH circuit (°C)"}},
	24:  oTValue{[]string{"room_temperature"}, cTypeF8_8, cTypeNone, []string{"Current sensed room temperature (°C)"}},
	25:  oTValue{[]string{"boiler_water_temp"}, cTypeF8_8, cTypeNone, []string{"Flow water temperature from boiler (°C)"}},
	26:  oTValue{[]string{"dhw_temperature"}, cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature (°C)"}},
	27:  oTValue{[]string{"outside_temperature"}, cTypeF8_8, cTypeNone, []string{"Outside air temperature (°C)"}},
	28:  oTValue{[]string{"return_water_temperature"}, cTypeF8_8, cTypeNone, []string{"Return water temperature to boiler (°C)"}},
	29:  oTValue{[]string{"solar_storage_temperature"}, cTypeF8_8, cTypeNone, []string{"Solar storage temperature (°C)"}},
	30:  oTValue{[]string{"solar_collector_temperature"}, cTypeS16, cTypeNone, []string{"Solar collector temperature (°C)"}},
	31:  oTValue{[]string{"flow_temperature_ch2"}, cTypeF8_8, cTypeNone, []string{"Flow water temperature of the second central heating circuit"}},
	32:  oTValue{[]string{"dhw2_temperature"}, cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature 2 (°C)"}},
	33:  oTValue{[]string{"exhaust_temperature"}, cTypeS16, cTypeNone, []string{"Exhaust temperature (°C)"}},
	48:  oTValue{[]string{"dhwsetpoint_upper_bound", "dhwsetpoint_lower_bound"}, cTypeS8, cTypeS8, []string{"Upper bound for adjustment of DHW setp (°C)", "Lower bound for adjustment of DHW setp (°C)"}},
	49:  oTValue{[]string{"max_chsetp_upper_bound", "max_chsetp_lower_bound"}, cTypeS8, cTypeS8, []string{"Upper bound for adjustment of maxCHsetp (°C)", "Lower bound for adjustment of maxCHsetp (°C)"}},
	56:  oTValue{[]string{"dhw_setpoint"}, cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature setpoint (°C)"}},
	57:  oTValue{[]string{"max_ch_water_setpoint"}, cTypeF8_8, cTypeNone, []string{"Maximum allowable CH water setpoint (°C)"}},
	100: oTValue{[]string{"manual_setpoint_overrules_remote_setpoint", "program_change_setpoint_overrides_remote_setpoint", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"}, cTypeNone, cTypeFlag8, []string{"Manual change priority [0 = disable overruling remote setpoint by manual setpoint change, 1 = enable overruling remote setpoint by manual setpoint change]", "Program change priority [0 = disable overruling remote setpoint by program setpoint change, 1 = enable overruling remote setpoint by program setpoint change]", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"}},
	115: oTValue{[]string{"oem_diagnostic_code"}, cTypeU16, cTypeNone, []string{"OEM-specific diagnostic/service code"}},
	116: oTValue{[]string{"burner_starts"}, cTypeU16, cTypeNone, []string{"Number of starts burner"}},
	117: oTValue{[]string{"ch_pump_starts"}, cTypeU16, cTypeNone, []string{"Number of starts CH pump"}},
	118: oTValue{[]string{"dhw_pump/valve_starts"}, cTypeU16, cTypeNone, []string{"Number of starts DHW pump/valve"}},
	119: oTValue{[]string{"dhw_burner_starts"}, cTypeU16, cTypeNone, []string{"Number of starts burner in DHW mode"}},
	120: oTValue{[]string{"burner_operation_hours"}, cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation (i.e.flame on)"}},
	121: oTValue{[]string{"ch_pump_operation_hours"}, cTypeU16, cTypeNone, []string{"Number of hours that CH pump has been running"}},
	122: oTValue{[]string{"dhw_pump/valve_operation_hours"}, cTypeU16, cTypeNone, []string{"Number of hours that DHW pump has been running or DHW valve has been opened"}},
	123: oTValue{[]string{"dhw_burner_operation_hours"}, cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation during DHW mode"}},
	124: oTValue{[]string{"opentherm_version_master"}, cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the master"}},
	125: oTValue{[]string{"opentherm_version_slave"}, cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the slave"}},
	126: oTValue{[]string{"master_product_version_number", "master_product_type"}, cTypeU8, cTypeU8, []string{"The master device product version number as defined by the manufacturer", "The master device product type as defined by the manufacturer"}},
	127: oTValue{[]string{"slave_product_version_number", "slave_product_type"}, cTypeU8, cTypeU8, []string{"The slave device product version number as defined by the manufacturer", "The slave device product type as defined by the manufacturer"}},
}

var testMessage = []string{"T80000200",
	"B40000200",
	"T10011B00",
	"BD0011B00",
	"T00110000",
}

func bytesToUInt(in []byte) uint16 {
	var result uint16 = 0
	for _, v := range in {
		result <<= 8
		result += uint16(v)
	}
	return result
}

func decodeF8_8(in []byte) string {
	// fmt.Println("decoding ", in)
	return fmt.Sprintf("%.2f", bytesToFloat(in))
}

func bytesToFloat(in []byte) float64 {
	// fmt.Println("decoding ", in)
	return float64(in[0]) + float64(in[1])/256
}

func byteToBool(in byte, bitPosition byte) bool {
	// fmt.Printf("flags % 08b \n", in)
	// fmt.Printf("mask  % 08b %d\n", (1 << bitPosition), bitPosition)
	isFlagSet := (in&(1<<bitPosition) > 0)
	return isFlagSet
}

func decodeFlag8(in byte, bitPosition byte) string {
	var result = "0"
	if byteToBool(in, bitPosition) {
		result = "1"
	}
	return result
}

func getMessageType(msg string) uint8 {
	var msgType uint8
	v, err := hex.DecodeString(msg[1:3])
	//	fmt.Println("decoding type from ", v[0])
	if err != nil {
		log.Printf("%v\n", err.Error())
		return cDataInvalid
	}
	msgType = uint8((v[0] >> 4) & 7)
	return msgType
}

func decodeMessage(v []byte, types []uint8, text []string) []string {
	var output []string
	var offset = 0 // offset on lowbyte decoding is 1 for most types, exception being cTypeFlag8 and cTypeU8WDT

	for index, valueType := range types {
		switch valueType {
		case cTypeFlag8:
			for i := 0; i < 7; i++ {
				output = append(output, fmt.Sprintf("%s=%s", text[i+offset], decodeFlag8(v[2], byte(i))))
			}
			offset += 8 // after decoding flags the next decoder should start with an offset of 8
		case cTypeF8_8:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], decodeF8_8(v[2:4])))
		case cTypeU16:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], bytesToUInt(v[2:4])))
		case cTypeS16:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], int16(bytesToUInt(v[2:4]))))
		case cTypeU8:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], bytesToUInt(v[2+index:3+index])))
			offset += 1 // after decoding an 8 bit number the next decoder should start with an offset of 1
		case cTypeS8:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], int8(bytesToUInt(v[2+index:3+index]))))
			offset += 1 // after decoding an 8 bit number the next decoder should start with an offset of 1
		case cTypeU8WDT:
			output = append(output, fmt.Sprintf("%s=%v", text[offset], v[2]>>5))   // top 3 bits
			output = append(output, fmt.Sprintf("%s=%v", text[offset+1], v[2]&31)) // bottom 5 bits
			offset += 1                                                            // after decoding this type the next decoder should start with an offset of 1
		case cTypeNone:
		default:
			log.Println("unknown type:", valueType)
		}
	}
	return output
}

func decodeLineProtocol(msg string) string {
	var output string
	const influxMeasurementName = "otgw"

	if isValidMsg(msg) {
		v, err := hex.DecodeString(msg[1:9])
		if err != nil {
			log.Printf("%v\n", err.Error())
			return output
		}
		msgID := v[1]
		decoder, exists := decoderMapReadable[msgID]
		if exists {
			fields := decodeMessage(v, []byte{decoder.highByteType, decoder.lowByteType}, decoder.fields)
			output = influxMeasurementName
			for _, field := range fields {
				output += " " + field
			}
			output += " " + fmt.Sprint(time.Now().UnixNano())
		}
	}
	return output
}

func decodeReadable(msg string) []string {
	var output []string

	if isValidMsg(msg) {
		v, err := hex.DecodeString(msg[1:9])
		if err != nil {
			log.Printf("%v\n", err.Error())
			return output
		}
		msgID := v[1]
		decoder, exists := decoderMapReadable[msgID]
		if exists {
			output = decodeMessage(v, []byte{decoder.highByteType, decoder.lowByteType}, decoder.descriptions)
		}
	}
	return output
}

func isValidMsg(msg string) bool {
	var valid = true

	valid = valid && (len(msg) == cOTGWmsgLength)
	valid = valid && (msg[0:1] == "T" || msg[0:1] == "B")

	if !valid {
		log.Println("Received invalid message:", msg)
	}

	return valid
}

func isDecodableMsgType(msg string) bool {
	otType := getMessageType(msg)
	if otType == cDataInvalid ||
		otType == cUnknownDataID ||
		otType == cInvalidData {
		log.Println("OT message contains invalid or unkonw data type:", msg)
	}
	// only the acknowledgements are worth decoding
	return (otType == cReadAck || otType == cWriteAck)
}
