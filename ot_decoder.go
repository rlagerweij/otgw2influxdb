package main

import "fmt"
import "strings"
import "encoding/hex"
import "strconv"

type openthermMessage struct {
	message []byte
}

const (
	cTypeFlag8 = 1 	// byte composed of 8 single-bit flags
	cTypeU8 = 2 	// unsigned 8-bit integer 0 .. 255
	cTypeS8 = 3		// signed 8-bit integer -128 .. 127 (two’s compliment)
	cTypeF8_8 = 4	// signed fixed point value : 1 sign bit, 7 integer bit, 8 fractional bits (two’s compliment ie. the LSB of the 16bit binary number represents 1/256 flag8 byte composed of 8 single-bit flags
	cTypeU16 = 5 	// unsigned 16-bit integer 0..65535
	cTypeS16 = 6 	// signed 16-bit integer -32768..32767
)

const (
	cReadData	= 0
	cWriteData	= 1
	cInvalidData	= 2
	cReserved	= 3
	cReadAck	= 4
	cWriteAck	= 5
	cDataInvalid	= 6
	cUnknownDataID	= 7
)

type oTValue struct {
	name		string
	highByteType   uint8
	lowByteType    uint8
	descriptions []string
}

var decoderMap = map[uint8]oTValue {
	0: oTValue{ "Status" , cTypeFlag8, cTypeFlag8, []string{ "CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved", }},
}

func main() {
	fmt.Println("Test")
}
