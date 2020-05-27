package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

var (
	sha1ver   string = "testing" // sha1 revision used to build the program
	buildTime string = "testing" // when the executable was built
)

func checkErrorLog(err error) {
	if err != nil {
		log.Printf("%v\n", err.Error())
	}
}

func checkErrorFatal(err error) {
	if err != nil {
		log.Printf("%v\n", err.Error())
		os.Exit(1)
	}
}

var addr = "10.0.0.130:6638"

func main() {

	log.Printf("Starting program (version: %s / build time: %s )\n", sha1ver, buildTime)

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", addr)

	checkErrorFatal(err)

	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		log.Print("Message from OTGW: " + message)
		if isValidMsg(message) && isDecodableMsgType(message) {
			readable := decodeReadable(message)
			for _, line := range readable {
				fmt.Println(line)
			}
			// fmt.Println(decodeLineProtocol(message))
		}
	}

}
