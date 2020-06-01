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

var influxWriteURL = "http://%s:%s/api/v2/write?bucket=%s&precision=s"

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
		parts := strings.SplitN(line, "#", 2) // split off any comments
		if strings.Contains(parts[0], "=") {
			kv := strings.SplitN(parts[0], "=", 2)
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
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

func sendToInfluxDB(out chan string) {

	influxURL := fmt.Sprintf(influxWriteURL,
		config["influxIP"],
		config["influxPort"],
		config["influxBucket"])

	for {
		lp := <-out
		dbBuffer += lp
		dbBufferCount++
		if dbBufferCount >= dbBufferMaxCount {
			client := &http.Client{}
			req, err := http.NewRequest("POST", influxURL, bytes.NewBufferString(dbBuffer))
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
}

func readMessagesFromOTGW(c chan string) {

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", config["OTGWaddress"])

	checkErrorFatal(err)

	for {
		msgIn, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Println(err)
		} else {
			if len(c) == cap(c) {
				dumpValue := <-c
				dumpValue += "now used" // go insists that we use values we declare
			}
			c <- msgIn
		}
	}
}

func main() {

	readConfig()
	log.Printf("Starting program (version: %s / build time: %s )\n", sha1ver, buildTime)

	receiveMessages := make(chan string, 10)
	sendMessages := make(chan string, 10)

	go readMessagesFromOTGW(receiveMessages)
	go sendToInfluxDB(sendMessages)

	for {
		message := <-receiveMessages
		// log.Print("Message from OTGW: " + message)
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
					// fmt.Print(lp)
					sendMessages <- lp
				}
			}
		}
	}
}
