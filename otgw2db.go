package main

import (
	"bufio"
	"bytes"
	"container/list"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

const dbBufferMaxCount = 20 // number of influx points to collect before sending them to the database

var influxWriteURL = "http://%s:%s/api/v2/write?bucket=%s&precision=s"

var logVerbose = log.New(ioutil.Discard, "", log.Ldate|log.Ltime)
var verboseFlagSet = false

var maxOtgwReconnectDelay = 600 // max delay in seconds for exponential back-off

func readConfig(fn string) {
	config = make(map[string]string)
	file, err := os.Open(fn)

	if err != nil {
		fmt.Println("no config files found. Please rename the supplied otgw2db.example.cfg file to otgw2db.cfg and adjust its contents.")
		log.Fatal("OS: ", err)
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
}

func sendToInfluxBuffer(out chan string) {
	var msgWritten = 0

	for {
		lp := <-out
		dbBuffer += lp
		dbBufferCount++
		logVerbose.Print("Added message ", dbBufferCount, " to the buffer:", lp)
		if dbBufferCount >= dbBufferMaxCount {
			err := sendToInfluxDB(dbBuffer)
			if err != nil {
				log.Println("Could not submit data to influxdb. Dropping data points")
			} else {
				msgWritten += dbBufferCount
				logVerbose.Printf("Submitted %v points to influxdb. total points written: %v\n", dbBufferCount, msgWritten)
			}
			dbBuffer = ""
			dbBufferCount = 0
		}
	}
}

func sendToInfluxDB(postBody string) error {

	influxURL := fmt.Sprintf(influxWriteURL,
		config["influxIP"],
		config["influxPort"],
		config["influxBucket"])

	client := &http.Client{}
	req, err := http.NewRequest("POST", influxURL, bytes.NewBufferString(postBody))
	if err != nil {
		log.Println("Creating http request failed: ", err.Error())
	}
	// log.Println(req)
	req.Header.Add("Authentication", fmt.Sprintf("Token %s:%s", config["influxUser"], config["influxPass"]))
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Http POST to influxdb failed: ", err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		log.Println("Http POST to influxdb returned status:", resp.Status)
		err = errors.New("Database does not accept data. Check settings")
	}
	return err
}

func startRelayListener(c chan net.Conn) {

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", config["relay_tcp_port"]))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Relay client connection error: ", err)
		} else {
			c <- conn
		}
	}
}

func sendRelayMessages(m chan string, c chan net.Conn) {
	var currentMsg string

	conns := list.New()

	for {
		select {
		case conn := <-c:
			{
				log.Printf("Relay client connected from: %s\n", conn.RemoteAddr().String())
				conns.PushBack(conn)
			}
		case currentMsg = <-m:
			for con := conns.Front(); con != nil; con = con.Next() {
				conItem := con.Value.(net.Conn)
				conItem.SetWriteDeadline(time.Now().Add(time.Second * 1))
				_, err := conItem.Write([]byte(currentMsg))
				if err != nil {
					log.Println("Relay error writing message:", err)
					conns.Remove(con)
				}
			}
		default:
			// include default to make the above non-blocking
			time.Sleep(time.Millisecond * 10) // add small delay to reduce cpu usage
		}
	}
}

func readMessagesFromOTGW(c chan string) {

	var connSuccess = false // used to indicate whether there has ever been a successful connection
	var connRetryCounter = 0
	var otgwReconnectDelay = 0
	var readErrorCount = 0

	for {
		d := net.Dialer{Timeout: 2 * time.Second}
		conn, err := d.Dial("tcp", config["OTGWaddress"])

		if err != nil {
			connRetryCounter++
			log.Println("Connection to otgw could not be established. Attempt ", connRetryCounter)
			if (connSuccess == false) && (connRetryCounter >= 3) {
				log.Fatal("Aborting program. Check your settings in otgw2db.cfg\n") // abort after 3 tries if there has not previously been a connection
			} else {
				time.Sleep(time.Second * time.Duration(otgwReconnectDelay))

				// exponential back-off on reconnecting to the OTGW
				if otgwReconnectDelay < maxOtgwReconnectDelay {
					otgwReconnectDelay = (1 << connRetryCounter)
				} else {
					otgwReconnectDelay = maxOtgwReconnectDelay
				}

				continue
			}
		} else {
			connSuccess = true

			// reset counters
			connRetryCounter = 0
			otgwReconnectDelay = 0
			readErrorCount = 0

			log.Println("Succesfully connected to OTGW at: ", conn.RemoteAddr())
		}

		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 10))
			msgIn, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				readErrorCount++
				log.Println("Error reading from otgw (count ", readErrorCount, "): ", err)
				if (err == io.EOF) || (readErrorCount > 5) {
					log.Println("Connection has timed out or was closed by otgw")
					break
				}
			} else {
				readErrorCount = 0
				if len(c) == cap(c) {
					_ = <-c //	dump value from channel
				}
				c <- msgIn
			}
		}
	}
}

func influxTest() bool {

	err := sendToInfluxDB("")
	if err != nil {
		log.Println("Influxdb test error: ", err)
		return false
	}
	return true
}

func main() {
	log.Printf("OTGW2DB - starting program (version: %s / build time: %s )\n", sha1ver, buildTime)
	flag.BoolVar(&verboseFlagSet, "v", false, ": set logging to verbose. Main use is testing, creates very large logs")
	flag.Parse()

	if verboseFlagSet {
		logVerbose.SetOutput(os.Stdout)
	}

	readConfig("otgw2db.cfg")

	if !influxTest() {
		log.Fatal("Could not connect to influxdb. Please check the settings in otgw2db.cfg")
	}

	receiveMessages := make(chan string, 10)
	sendMessages := make(chan string, 10)
	relayMessages := make(chan string, 10)
	relayClients := make(chan net.Conn)

	go readMessagesFromOTGW(receiveMessages)
	go sendToInfluxBuffer(sendMessages)
	go startRelayListener(relayClients)
	go sendRelayMessages(relayMessages, relayClients)

	for {
		message := <-receiveMessages
		logVerbose.Print("Message from OTGW: " + message)
		if len(relayMessages) == cap(relayMessages) {
			_ = <-relayMessages // dump value from channel
		}
		relayMessages <- message

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
					sendMessages <- lp
				}
			}
		}
		time.Sleep(time.Millisecond * 10) // add small delay to the main loop to reduce cpu usage
	}
}
