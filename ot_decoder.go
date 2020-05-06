package main

import "fmt"
import "os"
import "net"
import "bufio"
import "time"
import "encoding/hex"

const cOTGWmsgLength = 11

const (
	cTypeFlag8 = 1 	// byte composed of 8 single-bit flags
	cTypeU8 = 2 	// unsigned 8-bit integer 0 .. 255
	cTypeS8 = 3		// signed 8-bit integer -128 .. 127 (two’s compliment)
	cTypeF8_8 = 4	// signed fixed point value : 1 sign bit, 7 integer bit, 8 fractional bits (two’s compliment ie. the LSB of the 16bit binary number represents 1/256 flag8 byte composed of 8 single-bit flags
	cTypeU16 = 5 	// unsigned 16-bit integer 0..65535
	cTypeS16 = 6 	// signed 16-bit integer -32768..32767
	cTypeNone = 7
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

type openthermMessage struct {
	message []byte
}

type oTValue struct {
	name		string
	highByteType   uint8
	lowByteType    uint8
	descriptions []string
}

var decoderMapReadable = map[uint8]oTValue {
	0: oTValue{ "Status" , cTypeFlag8, cTypeFlag8, []string{ "CH enable", "DHW enable", "Cooling enable", "OTC active", "CH2 enable", "reserved", "reserved", "reserved", "Fault indication", "CH mode", "DHW mode", "Flame status", "Cooling status", "CH2 mode", "Diagnostic Event", "reserved", }},
	16: oTValue{ "room_setpoint", cTypeF8_8,cTypeNone, []string{"Current room temperature setpoint (°C)"}},
	17: oTValue{ "relative_modulation_level", cTypeF8_8,cTypeNone, []string{"Relative modulation level (%)"}},
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
	return float64(in[0])+float64(in[1])/256
}

func byteToBool(in byte, bitPosition byte) bool {
	return (in & (1<<bitPosition) > 0)
}

func getMessageType(msg string) uint8 {
	var msgType uint8
	v, err := hex.DecodeString(msg[1:9])
    checkError(err)
	msgType = uint8((v[0]>>4) & 7) 
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
			for i := 8; i < 15; i++ {
				fmt.Println(decoder.descriptions[i], byteToBool(v[3], byte(i)))
			}
		case cTypeNone:
		default:
			fmt.Println("unknown type")
		}
	}
	return output
} 

var testMessage = []string { 	"T80000200",
								"B40000200",
								"T10011B00",
								"BD0011B00",
								"T00110000", }

var addr = "10.0.0.130:6638"

func main() {

	fmt.Println("Starting program")

	d := net.Dialer{Timeout: 2 * time.Second}
    conn, err := d.Dial("tcp", addr)
    checkError(err)

	for{
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print("Message from OTGW: "+message)
		decodeReadable(message)
		fmt.Println()
	}

}
