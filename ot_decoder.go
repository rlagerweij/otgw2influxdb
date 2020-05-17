package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"time"
)

const cOTGWmsgLength = 11

const (
	cTypeU8    = 1 // unsigned 8-bit integer 0 .. 255
	cTypeU8WDT = 2 // byte representing Day of Week & Time of Day / HB : bits 7,6,5 : day of week / bits 4,3,2,1,0 : hours
	cTypeS8    = 3 // signed 8-bit integer -128 .. 127 (two’s compliment)
	cTypeF8_8  = 4 // signed fixed point value : 1 sign bit, 7 integer bit, 8 fractional bits (two’s compliment ie. the LSB of the 16bit binary number represents 1/256 flag8 byte composed of 8 single-bit flags
	cTypeU16   = 5 // unsigned 16-bit integer 0..65535
	cTypeS16   = 6 // signed 16-bit integer -32768..32767
	cTypeNone  = 7
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
	message []byte
}

type oTValue struct {
	name         string
	highByteType uint8
	lowByteType  uint8
	descriptions []string
}

var decoderMapReadable = map[uint8]oTValue{
	0:   oTValue{"status", cTypeFlag8, cTypeFlag8, []string{"CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved"}},
	1:   oTValue{"control_setpoint", cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler in degrees C"}},
	2:   oTValue{"master_configuration", cTypeNone, cTypeU8, []string{"MemberID code of the master"}},
	3:   oTValue{"slave_configuration", cTypeFlag8, cTypeU8, []string{"DHW present [ dhw not present, dhw is present ]", "Control type [ modulating, on/off ]", "Cooling config [ cooling not supported, cooling supported]", "DHW config [instantaneous or not-specified, storage tank]", "Master low-off&pump control function [allowed, not allowed]", "CH2 present [CH2 not present, CH2 present]", "reserved", "reserved", "reserved", "MemberID code of the slave"}},
	5:   oTValue{"application-specific fault flags", cTypeFlag8, cTypeU8, []string{"Service request [service not req’d, service required]", "Lockout-reset [ remote reset disabled, rr enabled]", "Low water press [no WP fault, water pressure fault]", "Gas/flame fault [ no G/F fault, gas/flame fault ]", "Air press fault [ no AP fault, air pressure fault ]", "Water over-temp[no OvT fault, over-temperat. Fault]", "reserved", "reserved", "OEM fault code u8 0..255 An OEM-specific fault/error code"}},
	7:   oTValue{"cooling_control_signal", cTypeF8_8, cTypeNone, []string{"Signal for cooling plant"}},
	8:   oTValue{"control_setpoint_2", cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler for circuit 2 in degrees C"}},
	9:   oTValue{"remote_override_room_setpoint", cTypeF8_8, cTypeNone, []string{"Remote override room setpoint (0 = No override)"}},
	10:  oTValue{"number_of_tsps ", cTypeU8, cTypeU8, []string{"Number of transparent-slave-parameter supported by the slave device"}},
	11:  oTValue{"tsp_index_no", cTypeU8, cTypeU8, []string{"Index number of following TSP", "Value of above referenced TSP"}},
	12:  oTValue{"size_of_fault_buffer ", cTypeU8, cTypeNone, []string{"The size of the fault history buffer"}},
	13:  oTValue{"FHB_entry_index_no.", cTypeU8, cTypeU8, []string{"Index number of following Fault Buffer entry", "Value of above referenced Fault Buffer entry"}},
	14:  oTValue{"maximum_relative_modulation_level_setting", cTypeF8_8, cTypeNone, []string{"Maximum relative boiler modulation level setting for sequencer and off-low&pump control applications (%)"}},
	15:  oTValue{"boiler_characteristics", cTypeU8, cTypeU8, []string{"Maximum boiler capacity (kW)", "Minimum modulation level (%)"}},
	16:  oTValue{"room_setpoint", cTypeF8_8, cTypeNone, []string{"Current room temperature setpoint (°C)"}},
	17:  oTValue{"relative_modulation_level", cTypeF8_8, cTypeNone, []string{"Relative modulation level (%)"}},
	18:  oTValue{"ch_water_pressure", cTypeF8_8, cTypeNone, []string{"Water pressure of the boiler CH circuit (bar)"}},
	19:  oTValue{"dhw_flow_rate", cTypeF8_8, cTypeNone, []string{"Water flow rate through the DHW circuit (l/min)"}},
	20:  oTValue{"weekday_time", cTypeU8WDT, cTypeU8, []string{"Day of the week (1=Monday)", "Hours", "Minutes"}},
	21:  oTValue{"date", cTypeU8, cTypeU8, []string{"Month", "Day of Month"}},
	22:  oTValue{"year", cTypeU16, cTypeNone, []string{"Year"}},
	23:  oTValue{"room_setpoint_ch2", cTypeF8_8, cTypeNone, []string{"Current room setpoint for 2nd CH circuit (°C)"}},
	24:  oTValue{"room_temperature", cTypeF8_8, cTypeNone, []string{"Current sensed room temperature (°C)"}},
	25:  oTValue{"boiler_water_temp", cTypeF8_8, cTypeNone, []string{"Flow water temperature from boiler (°C)"}},
	26:  oTValue{"dhw_temperature", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature (°C)"}},
	27:  oTValue{"outside_temperature", cTypeF8_8, cTypeNone, []string{"Outside air temperature (°C)"}},
	28:  oTValue{"return_water_temperature", cTypeF8_8, cTypeNone, []string{"Return water temperature to boiler (°C)"}},
	29:  oTValue{"solar_storage_temperature", cTypeF8_8, cTypeNone, []string{"Solar storage temperature (°C)"}},
	30:  oTValue{"solar_collector_temperature", cTypeS16, cTypeNone, []string{"Solar collector temperature (°C)"}},
	31:  oTValue{"flow_temperature_ch2", cTypeF8_8, cTypeNone, []string{"Flow water temperature of the second central heating circuit"}},
	32:  oTValue{"dhw2_temperature", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature 2 (°C)"}},
	33:  oTValue{"exhaust_temperature", cTypeS16, cTypeNone, []string{"Exhaust temperature (°C)"}},
	48:  oTValue{"dhwsetpoint_bounds", cTypeS8, cTypeS8, []string{"Upper bound for adjustment of DHW setp (°C)", "Lower bound for adjustment of DHW setp (°C)"}},
	49:  oTValue{"max_chsetp_bounds", cTypeS8, cTypeS8, []string{"Upper bound for adjustment of maxCHsetp (°C)", "Lower bound for adjustment of maxCHsetp (°C)"}},
	56:  oTValue{"dhw_setpoint", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature setpoint (°C)"}},
	57:  oTValue{"max_ch_water_setpoint", cTypeF8_8, cTypeNone, []string{"Maximum allowable CH water setpoint (°C)"}},
	100: oTValue{"remote_override_function", cTypeNone, cTypeFlag8, []string{"Manual change priority [0 = disable overruling remote setpoint by manual setpoint change, 1 = enable overruling remote setpoint by manual setpoint change]", "Program change priority [0 = disable overruling remote setpoint by program setpoint change, 1 = enable overruling remote setpoint by program setpoint change]", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"}},
	115: oTValue{"oem_diagnostic_code", cTypeU16, cTypeNone, []string{"OEM-specific diagnostic/service code"}},
	116: oTValue{"burner_starts", cTypeU16, cTypeNone, []string{"Number of starts burner"}},
	117: oTValue{"ch_pump_starts", cTypeU16, cTypeNone, []string{"Number of starts CH pump"}},
	118: oTValue{"dhw_pump/valve_starts", cTypeU16, cTypeNone, []string{"Number of starts DHW pump/valve"}},
	119: oTValue{"dhw_burner_starts", cTypeU16, cTypeNone, []string{"Number of starts burner in DHW mode"}},
	120: oTValue{"burner_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation (i.e.flame on)"}},
	121: oTValue{"ch_pump_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that CH pump has been running"}},
	122: oTValue{"dhw_pump/valve_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that DHW pump has been running or DHW valve has been opened"}},
	123: oTValue{"dhw_burner_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation during DHW mode"}},
	124: oTValue{"opentherm_version_master", cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the master"}},
	125: oTValue{"opentherm_version_slave", cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the slave"}},
	126: oTValue{"master_product_version number and type", cTypeU8, cTypeU8, []string{"The master device product version number as defined by the manufacturer", "The master device product type as defined by the manufacturer"}},
	127: oTValue{"slave_product_version number and type", cTypeU8, cTypeU8, []string{"The slave device product version number as defined by the manufacturer", "The slave device product type as defined by the manufacturer"}},
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(1)
	}
}

func bytesToUInt(in []byte) uint16 {
	var result uint16 = 0
	for _, v := range in {
		result <<= 8
		result += uint16(v)
	}
	return result
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

func getMessageType(msg string) uint8 {
	var msgType uint8
	v, err := hex.DecodeString(msg[1:3])
	//	fmt.Println("decoding type from ", v[0])
	checkError(err)
	msgType = uint8((v[0] >> 4) & 7)
	return msgType
}

func decodeReadable(msg string) []string {
	var output []string
	var lowByteOffset = 1 // offset on lowbyte decoding is 1 for most types, exception being cTypeFlag8 and cTypeU8WDT

	if len(msg) == cOTGWmsgLength {
		v, err := hex.DecodeString(msg[1:9])
		checkError(err)
		msgID := v[1]
		decoder := decoderMapReadable[msgID]

		switch decoder.highByteType {
		case cTypeFlag8:
			fmt.Println("High byte")
			lowByteOffset = cTypeFlag8 // constant value was set to required offset
			for i := 0; i < 7; i++ {
				output = append(output, fmt.Sprintln(decoder.descriptions[i], byteToBool(v[2], byte(i))))
			}
		case cTypeF8_8:
			output = append(output, fmt.Sprintln(decoder.descriptions[0], bytesToFloat(v[2:4])))
		case cTypeU16:
			output = append(output, fmt.Sprintln(decoder.descriptions[0], bytesToUInt(v[2:4])))
		case cTypeS16:
			output = append(output, fmt.Sprintln(decoder.descriptions[0], int16(bytesToUInt(v[2:4]))))
		case cTypeU8:
			output = append(output, fmt.Sprintln(decoder.descriptions[0], bytesToUInt(v[2:3])))
		case cTypeS8:
			output = append(output, fmt.Sprintln(decoder.descriptions[0], int8(bytesToUInt(v[2:3]))))
		case cTypeU8WDT:
			lowByteOffset = cTypeU8WDT                                              // constant value was set to required offset
			output = append(output, fmt.Sprintln(decoder.descriptions[0], v[2]>>5)) // top 3 bits
			output = append(output, fmt.Sprintln(decoder.descriptions[1], v[2]&31)) // bottom 5 bits
		default:
			output = append(output, fmt.Sprintln("unknown type"))
		}

		switch decoder.lowByteType {
		case cTypeFlag8:
			fmt.Println("Low byte")
			//			fmt.Printf("decode flags % 08b \n", v[3])
			for i := 0; i < 7; i++ {
				output = append(output, fmt.Sprintln(decoder.descriptions[i+lowByteOffset], byteToBool(v[3], byte(i))))
			}
		case cTypeU8:
			output = append(output, fmt.Sprintln(decoder.descriptions[lowByteOffset], bytesToUInt(v[3:4])))
		case cTypeS8:
			output = append(output, fmt.Sprintln(decoder.descriptions[lowByteOffset], int8(bytesToUInt(v[3:4]))))
		case cTypeNone:
		default:
			output = append(output, fmt.Sprintln("unknown type"))
		}
	}
	return output
}

var testMessage = []string{"T80000200",
	"B40000200",
	"T10011B00",
	"BD0011B00",
	"T00110000"}

var addr = "10.0.0.130:6638"

func main() {

	fmt.Println("Starting program")

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", addr)
	checkError(err)

	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print("Message from OTGW: " + message)
		if (len(message) == cOTGWmsgLength) && (getMessageType(message) == cReadAck || getMessageType(message) == cWriteAck) {
			fmt.Println("length message: ", len(message))
			readable := decodeReadable(message)
			for _, line := range readable {
				fmt.Print(line)
			}
		}
	}

}
