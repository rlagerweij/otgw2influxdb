package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var ( // these variable as set at build time, they do not belong in the config map
	sha1ver   string = "testing" // sha1 revision used to build the program
	buildTime string = "testing" // when the executable was built
)

var config map[string]string

var dbBuffer string
var dbBufferCount int

const dbBufferMaxCount = 10

func readConfig() {
	config = make(map[string]string)
	file, err := os.Open("ot_decoder.cfg")

	if err != nil {
		fmt.Println("no config files found. Please rename the supplied ot_decoder.example.cfg file to ot_decoder.cfg and adjust its contents.")
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			//	parts = strings.Split(parts[1], "#")
			val := strings.TrimSpace(strings.Split(parts[1], "#")[0])
			config[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

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

func sendToInfluxDB(lp string) {

	dbBuffer += lp
	dbBufferCount++
	if dbBufferCount >= dbBufferMaxCount {

		client := &http.Client{}
			req, err := http.NewRequest("POST", config["influxURL"], bytes.NewBufferString(dbBuffer))
		if err != nil {
			log.Println("creating http request failed: ", err.Error())
		}
		req.Header.Add("Authentication", fmt.Sprintf("Token %s:%s", config["influxUser"], config["influxPass"]))
		resp, err := client.Do(req)
		if err != nil {
			log.Println("http POST to influxdb failed: ", err.Error())
		}

		defer resp.Body.Close()

		if resp.StatusCode != 204 {
			log.Println("http POST to influxdb returned status:", resp.Status)
		}
		dbBuffer = ""
		dbBufferCount = 0
	}

}

func main() {

	readConfig()
	log.Printf("Starting program (version: %s / build time: %s )\n", sha1ver, buildTime)

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", config["OTGWaddress"])

	checkErrorFatal(err)

	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		//		log.Print("Message from OTGW: " + message)
		if isValidMsg(message) && isDecodableMsgType(message) {
			if strings.Contains(config["decode_readable"], "YES") {
				readable := decodeReadable(message)
				for _, line := range readable {
					fmt.Println(line)
				}
			}
			if strings.Contains(config["decode_line_protocol"], "YES") {
				lp := decodeLineProtocol(message)
				if len(lp) > 0 {
					fmt.Print(lp)
					// sendToInfluxDB(lp)
				}
			}
		}
	}
}
