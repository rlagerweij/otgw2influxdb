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
	cTypeNone  = 0
	cTypeU8    = 1 // unsigned 8-bit integer 0 .. 255
	cTypeU8WDT = 2 // byte representing Day of Week & Time of Day / HB : bits 7,6,5 : day of week / bits 4,3,2,1,0 : hours
	cTypeS8    = 3 // signed 8-bit integer -128 .. 127 (two’s compliment)
	cTypeF8_8  = 4 // signed fixed point value : 1 sign bit, 7 integer bit, 8 fractional bits (two’s compliment ie. the LSB of the 16bit binary number represents 1/256 flag8 byte composed of 8 single-bit flags
	cTypeU16   = 5 // unsigned 16-bit integer 0..65535
	cTypeS16   = 6 // signed 16-bit integer -32768..32767
	cTypeFlag8 = 7 // byte composed of 8 single-bit flags
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
	0: oTValue{"Status", cTypeFlag8, cTypeFlag8, []string{"CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved"}},
	1: oTValue{"Control_setpoint", cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler in degrees C"}},
	2: oTValue{"Master_configuration", cTypeNone, cTypeU8, []string{"MemberID code of the master"}},
	3: oTValue{"Slave_configuration", cTypeFlag8, cTypeU8, []string{"DHW present [ dhw not present, dhw is present ]", "Control type [ modulating, on/off ]", "Cooling config [ cooling not supported, cooling supported]", "DHW config [instantaneous or not-specified,	storage tank]", "Master low-off&pump control function [allowed,	not allowed]", "CH2 present [CH2 not present, CH2 present]", "reserved", "reserved", "reserved", "MemberID code of the slave"}},
	5:   oTValue{"Application-specific fault flags", cTypeFlag8, cTypeU8, []string{"Service request [service not req’d, service required]", "Lockout-reset [ remote reset disabled, rr enabled]", "Low water press [no WP fault, water pressure fault]", "Gas/flame fault [ no G/F fault, gas/flame fault ]", "Air press fault [ no AP fault, air pressure fault ]", "Water over-temp[ no OvT fault, over-temperat. Fault]", "reserved", "reserved", " OEM fault code u8 0..255 An OEM-specific fault/error code"}},
	7:   oTValue{"Cooling_control_signal", cTypeF8_8, cTypeNone, []string{"Signal for cooling plant"}},
	8:   oTValue{"Control_setpoint_2", cTypeF8_8, cTypeNone, []string{"Temperature setpoint for the supply from the boiler for circuit 2 in degrees C"}},
	9:   oTValue{"Remote_override_room_setpoint", cTypeF8_8, cTypeNone, []string{"Remote override room setpoint (0 = No override)"}},
	10:  oTValue{"Number_of_TSPs ", cTypeU8, cTypeU8, []string{"Number of transparent-slave-parameter supported by the slave device"}},
	11:  oTValue{"TSP_index_no", cTypeU8, cTypeU8, []string{"Index number of following TSP", "Value of above referenced TSP"}},
	12:  oTValue{"Size of Fault Buffer ", cTypeU8, cTypeNone, []string{"The size of the fault history buffer"}},
	13:  oTValue{"FHB-entry index no.", cTypeU8, cTypeU8, []string{"Index number of following Fault Buffer entry", "Value of above referenced Fault Buffer entry"}},
	14:  oTValue{"Maximum_relative_modulation_level_setting", cTypeF8_8, cTypeNone, []string{"Maximum relative boiler modulation level setting for sequencer and off-low&pump control applications (%)"}},
	15:  oTValue{"boiler_characteristics", cTypeU8, cTypeU8, []string{"Maximum boiler capacity (kW)", "Minimum modulation level (%)"}},
	16:  oTValue{"room_setpoint", cTypeF8_8, cTypeNone, []string{"Current room temperature setpoint (°C)"}},
	17:  oTValue{"relative_modulation_level", cTypeF8_8, cTypeNone, []string{"Relative modulation level (%)"}},
	18:  oTValue{"CH_water_pressure", cTypeF8_8, cTypeNone, []string{"Water pressure of the boiler CH circuit (bar)"}},
	19:  oTValue{"DHW_flow_rate", cTypeF8_8, cTypeNone, []string{"Water flow rate through the DHW circuit (l/min)"}},
	20:  oTValue{"Weekday_time", cTypeU8WDT, cTypeU8, []string{"Day of the week (1=Monday)", "Hours", "Minutes"}},
	21:  oTValue{"Date", cTypeU8, cTypeU8, []string{"Month", "Day of Month"}},
	22:  oTValue{"Year", cTypeU16, cTypeNone, []string{"Year"}},
	23:  oTValue{"Room_Setpoint_CH2", cTypeF8_8, cTypeNone, []string{"Current room setpoint for 2nd CH circuit (°C)"}},
	24:  oTValue{"Room_temperature", cTypeF8_8, cTypeNone, []string{"Current sensed room temperature (°C)"}},
	25:  oTValue{"Boiler_water_temp", cTypeF8_8, cTypeNone, []string{"Flow water temperature from boiler (°C)"}},
	26:  oTValue{"DHW_temperature", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature (°C)"}},
	27:  oTValue{"Outside_temperature", cTypeF8_8, cTypeNone, []string{"Outside air temperature (°C)"}},
	28:  oTValue{"Return_water_temperature", cTypeF8_8, cTypeNone, []string{"Return water temperature to boiler (°C)"}},
	29:  oTValue{"Solar_storage_temperature", cTypeF8_8, cTypeNone, []string{"Solar storage temperature (°C)"}},
	30:  oTValue{"Solar_collector_temperature", cTypeS16, cTypeNone, []string{"Solar collector temperature (°C)"}},
	31:  oTValue{"Flow_temperature_CH2", cTypeF8_8, cTypeNone, []string{"Flow water temperature of the second central heating circuit"}},
	32:  oTValue{"DHW2_temperature", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature 2 (°C)"}},
	33:  oTValue{"Exhaust_temperature", cTypeS16, cTypeNone, []string{"Exhaust temperature (°C)"}},
	48:  oTValue{"DHWsetpoint_bounds", cTypeS8, cTypeS8, []string{"Upper bound for adjustment of DHW setp (°C)", "Lower bound for adjustment of DHW setp (°C)"}},
	49:  oTValue{"max_CHsetp_bounds", cTypeS8, cTypeS8, []string{"Upper bound for adjustment of maxCHsetp (°C)", "Lower bound for adjustment of maxCHsetp (°C)"}},
	56:  oTValue{"DHW_setpoint", cTypeF8_8, cTypeNone, []string{"Domestic hot water temperature setpoint (°C)"}},
	57:  oTValue{"max_CH_water_setpoint", cTypeF8_8, cTypeNone, []string{"Maximum allowable CH water setpoint (°C)"}},
	100: oTValue{"Remote_override_function", cTypeNone, cTypeFlag8, []string{"Manual change priority [0 = disable overruling remote setpoint by manual setpoint change, 1 = enable overruling remote setpoint by manual setpoint change]", "Program change priority [0 = disable overruling remote setpoint by program setpoint change, 1 = enable overruling remote setpoint by program setpoint change]", "reserved", "reserved", "reserved", "reserved", "reserved", "reserved"}},
	115: oTValue{"OEM_diagnostic_code", cTypeU16, cTypeNone, []string{"OEM-specific diagnostic/service code"}},
	116: oTValue{"Burner_starts", cTypeU16, cTypeNone, []string{"Number of starts burner"}},
	117: oTValue{"CH_pump_starts", cTypeU16, cTypeNone, []string{"Number of starts CH pump"}},
	118: oTValue{"DHW_pump/valve_starts", cTypeU16, cTypeNone, []string{"Number of starts DHW pump/valve"}},
	119: oTValue{"DHW_burner_starts", cTypeU16, cTypeNone, []string{"Number of starts burner in DHW mode"}},
	120: oTValue{"Burner_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation (i.e.flame on)"}},
	121: oTValue{"CH_pump_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that CH pump has been running"}},
	122: oTValue{"DHW_pump/valve_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that DHW pump has been running or DHW valve has been opened"}},
	123: oTValue{"DHW_burner_operation_hours", cTypeU16, cTypeNone, []string{"Number of hours that burner is in operation during DHW mode"}},
	124: oTValue{"OpenTherm_version_Master", cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the master"}},
	125: oTValue{"OpenTherm_version_Slave", cTypeF8_8, cTypeNone, []string{"The implemented version of the OpenTherm Protocol Specification in the slave"}},
	126: oTValue{"Master_product_version number and type", cTypeU8, cTypeU8, []string{"The master device product version number as defined by the manufacturer", "The master device product type as defined by the manufacturer"}},
	127: oTValue{"Slave_product_version number and type", cTypeU8, cTypeU8, []string{"The slave device product version number as defined by the manufacturer", "The slave device product type as defined by the manufacturer"}},
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(1)
	}
}

func bytesToInt(in []byte) int64 {
	var result int64
	for _, v := range in {
		result <<= 8
		result += int64(v)
	}
	return result
}

func bytesToFloat(in []byte) float64 {
	fmt.Println("decoding ", in)
	return float64(in[0]) + float64(in[1])/256
}

func byteToBool(in byte, bitPosition byte) bool {
	return (in & (1<<bitPosition) > 0)
}

func getMessageType(msg string) uint8 {
	var msgType uint8
	v, err := hex.DecodeString(msg[1:9])
	checkError(err)
	msgType = uint8((v[0] >> 4) & 7)
	return msgType
}

func decodeReadable(msg string) []string {
	var output []string

	fmt.Println("length ", len(msg))

	if len(msg) == cOTGWmsgLength {
		fmt.Println(msg[1:9])
		v, err := hex.DecodeString(msg[1:9])
		checkError(err)
		msgID := v[1]
		fmt.Println(msgID)
		decoder := decoderMapReadable[msgID]

		switch decoder.highByteType {
		case cTypeFlag8:
			fmt.Printf("decode flags % 08b \n", v[2])
			for i := 0; i < 7; i++ {
				fmt.Println(decoder.descriptions[i], byteToBool(v[2], byte(i)))
			}
		case cTypeF8_8:
			fmt.Println("decode float from ", v)
			fmt.Println(decoder.descriptions[0], bytesToFloat(v[2:4]))

		default:
			fmt.Println("unknown type")
		}

		switch decoder.lowByteType {
		case cTypeFlag8:
			fmt.Printf("decode flags % 08b \n", v[3])
			for i := 0; i < 7; i++ {
				fmt.Println(decoder.descriptions[i+lowByteOffset], byteToBool(v[3], byte(i)))
			}
		case cTypeNone:
		default:
			fmt.Println("unknown type")
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
		decodeReadable(message)
		fmt.Println()
	}

}
