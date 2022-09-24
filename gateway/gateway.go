package gateway

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/haplesspanda/haplessbot/commands"
	"github.com/haplesspanda/haplessbot/constants"
	"github.com/haplesspanda/haplessbot/types"
)

var heartbeatTimer *time.Timer
var lastSequence *int
var sequenceLock = sync.Mutex{}
var sessionId *string

func init() {
	rand.Seed(time.Now().UnixNano())
}

func StartConnection() {
	gatewayUrl := getGatewayUrl(true)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	connect(gatewayUrl, interrupt)
}

func connect(url string, interrupt chan os.Signal) {
	reconnect := false

	for {
		log.Printf("Connecting to %s, reconnect=%t", url, reconnect)
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Fatalf("Dial error: %s", err)
		}

		reconnectChannel := make(chan struct{})

		go func() {
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Printf("Read error: %s", err)
					// TODO: Should only reconnect on some errors here
					close(reconnectChannel)
					return
				}
				log.Printf("received raw: %s", message)

				type GenericMessage struct {
					Op int     `json:"op"`
					D  any     `json:"d"`
					S  *int    `json:"s"`
					T  *string `json:"t"`
				}
				var parsedResponse GenericMessage
				json.Unmarshal(message, &parsedResponse)

				log.Printf("Parsed response as: %v", parsedResponse)

				switch parsedResponse.Op {
				case 0: // Event dispatch (everything else)
					type InteractionCreateMessage struct {
						Opt int                            `json:"op"`
						D   types.InteractionCreateDetails `json:"d"`
						T   string                         `json:"t"`
						S   int                            `json:"s"`
					}

					var parsedMessage InteractionCreateMessage
					json.Unmarshal(message, &parsedMessage)

					log.Printf("Parsed event dispatch message as %v", parsedMessage)

					if parsedMessage.T == "READY" {
						type ReadyMessageDetails struct {
							SessionId string `json:"session_id"`
						}

						type ReadyMessage struct {
							Op int                 `json:"op"`
							D  ReadyMessageDetails `json:"d"`
							T  string              `json:"t"`
							S  string              `json:"s"`
						}

						var readyMessage ReadyMessage
						json.Unmarshal(message, &readyMessage)

						log.Printf("Parsed ready message as %v", readyMessage)
						sessionId = &readyMessage.D.SessionId
					} else if parsedMessage.T == "INTERACTION_CREATE" {
						commands.RunInteractionCallback(parsedMessage.D)
					}
					setSequence(&parsedMessage.S)
				case 1: // Heartbeat
					type HeartbeatMessageRecv struct {
						Op int `json:"op"`
						D  any `json:"d"`
					}
					var parsedHeartbeatMessage HeartbeatMessageRecv
					json.Unmarshal(message, &parsedHeartbeatMessage)

					log.Printf("Parsed heartbeat message as %v", parsedHeartbeatMessage)
					// Immediate response.
					writeHeartbeat(c)
				case 7: // Reconnect
					close(reconnectChannel)
					return
				case 10: // Hello
					type HelloMessage struct {
						Op int `json:"op"`
						D  struct {
							HeartbeatInterval int `json:"heartbeat_interval"`
						} `json:"d"`
					}
					var parsedHelloMessage HelloMessage
					json.Unmarshal(message, &parsedHelloMessage)

					log.Printf("Parsed hello response as: %v", parsedHelloMessage)

					go heartbeatScheduler(c, parsedHelloMessage.D.HeartbeatInterval)
					if reconnect {
						resume(c)
					} else {
						identify(c)
					}
				case 11: // Heartbeat ack
					type HeartbeatAckMessage struct {
						Op int `json:"op"`
					}
					var parsedHeartbeatAckMessage HeartbeatAckMessage
					json.Unmarshal(message, &parsedHeartbeatAckMessage)

					log.Printf("Parsed heartbeat ack response as %v", parsedHeartbeatAckMessage)
					// TODO: Track acks and reconnect if not receiving any
				}

			}
		}()

		select {
		case <-reconnectChannel:
			c.Close()
			reconnect = true
			continue
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("write close error: %s", err)
				return
			}
			<-time.After(time.Second)
			return
		}
	}
}

func identify(conn *websocket.Conn) {
	type Properties struct {
		Os      string `json:"os"`
		Browser string `json:"browser"`
		Device  string `json:"device"`
	}

	type IdentifyMessageDetails struct {
		Token      string     `json:"token"`
		Properties Properties `json:"properties"`
	}

	type IdentifyMessage struct {
		Op      int                    `json:"op"`
		D       IdentifyMessageDetails `json:"d"`
		Intents int                    `json:"intents"`
	}

	identifyMessage := new(IdentifyMessage)
	identifyMessage.Op = 2
	identifyMessage.D = IdentifyMessageDetails{
		Token: constants.TokenId,
		Properties: Properties{
			Os:      "Windows",
			Browser: "haplessbot",
			Device:  "haplessbot",
		},
	}
	identifyMessage.Intents = 0
	write(conn, identifyMessage)
}

func resume(conn *websocket.Conn) {
	type ResumeMessageDetails struct {
		Token     string  `json:"token"`
		SessionId *string `json:"session_id"`
		Sequence  *int    `json:"seq"`
	}

	type ResumeMessage struct {
		Op int                  `json:"op"`
		D  ResumeMessageDetails `json:"d"`
	}

	sequence := getSequence()

	resumeMessage := new(ResumeMessage)
	resumeMessage.Op = 6
	resumeMessage.D = ResumeMessageDetails{
		Token:     constants.TokenId,
		SessionId: sessionId,
		Sequence:  sequence,
	}
	write(conn, resumeMessage)
}

func heartbeatScheduler(conn *websocket.Conn, intervalMillis int) {
	// Add jitter to first heartbeat.
	scheduleInterval := float64(intervalMillis) * rand.Float64()
	scheduleHeartbeat(conn, int(scheduleInterval))
	// Rest of heartbeats stay on the schedule.
	for {
		scheduleHeartbeat(conn, intervalMillis)
	}
}

func scheduleHeartbeat(conn *websocket.Conn, intervalMillis int) {
	if heartbeatTimer != nil {
		heartbeatTimer.Stop()
	}
	heartbeatTimer = time.NewTimer(time.Duration(intervalMillis) * time.Millisecond)
	defer heartbeatTimer.Stop()
	log.Printf("Scheduling heartbeat in %d millis", intervalMillis)
	<-heartbeatTimer.C
	writeHeartbeat(conn)
}

func setSequence(sequence *int) {
	sequenceLock.Lock()
	lastSequence = sequence
	sequenceLock.Unlock()
}

func getSequence() *int {
	sequenceLock.Lock()
	defer sequenceLock.Unlock()
	return lastSequence
}

func writeHeartbeat(conn *websocket.Conn) {
	sequence := getSequence()
	type heartbeat struct {
		Op int  `json:"op"`
		D  *int `json:"d"`
	}
	heartbeatJson := new(heartbeat)
	heartbeatJson.Op = 1
	heartbeatJson.D = sequence
	write(conn, heartbeatJson)
}

func write(conn *websocket.Conn, jsonMessage any) {
	formatJson, err := json.MarshalIndent(jsonMessage, "", "    ")
	if err != nil {
		panic(err)
	}
	log.Printf("Writing: %s", formatJson)
	err = conn.WriteJSON(jsonMessage)
	if err != nil {
		log.Printf("write err: %s", err)
		return
	}
}

func getGatewayUrl(useCache bool) string {
	if !useCache {
		return getCachedUrl()
	}

	// Lookup the latest.
	response, err := http.Get("https://discord.com/api/v10/gateway")
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var urlJson struct{ Url string }
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(body, &urlJson)
	if err != nil {
		panic(err)
	}
	return urlJson.Url
}

func getCachedUrl() string {
	// TODO: Cache more appropriately
	return "wss://gateway.discord.gg"
}
