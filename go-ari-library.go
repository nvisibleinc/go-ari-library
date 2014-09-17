package ari

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"log"
)

var (
	inCommand chan *Command
	inFlightCommands map[string]chan []byte
	commandChannel chan []byte
)

func init() {
	inFlightCommands = make(map[string]chan []byte)
}
// Event struct contains the events we pull off the websocket connection.
type Event struct {
	ServerID  string    `json:"server_id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	ARI_Body  string    `json:"ari_body"`
}

// Command struct contains the command we're passing back to ARI.
type Command struct {
	UniqueID	string	`json:"unique_id"`
	URL			string	`json:"url"`
	Method		string 	`json:"method"`
	Body		string	`json:"body"`
}

// UUID generates and returns a universally unique identifier.
func UUID() string {
	f, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

// InitProducer initializes a new message bus producer.
// The InitProducer uses the configuration to determine which message bus to
// connect to, and is thus message bus agnostic for the proxy and client.
func InitProducer(busType string, config interface{}, app string) chan []byte {
	var producer chan []byte
	switch busType {
	case "NSQ":
		// Start an NSQ producer
		producer = startNSQProducer(config, app)
	case "OSLO":
		// Start an OSLO producer
		log.Fatal("OSLO message bus producer is not yet implemented.")
	case "RABBITMQ":
		// Start a RabbitMQ producer
		log.Fatal("RABBITMQ message bus producer is not yet implemented.")
	default:
		log.Fatal("No bus type was specified for the producer that we recognize.")
	}
	return producer
}


func initMessageBus(app string, busType string, config interface{}) chan []byte {
	var busFeed chan []byte
	switch busType {
	case "NSQ":
		// Star NSQ Consumer
		busFeed = startNSQConsumer(config, app)
	default:
		log.Fatal("No bus type was specified for the consumer that we recognize")
	}
	return busFeed
}

// InitConsumer initializes a new message bus consumer
func InitConsumer(app string, busType string, config interface{}) chan *Event {
	// create channel to place parsed events onto
	parsedEvents := make(chan *Event)
	ProcessEvents(initMessageBus(app, busType, config), parsedEvents)
	return parsedEvents
}

func InitCommandProducer(busType string, config interface{}, app string) {
	commandChannel = InitProducer(busType, config, app)
}

// ProcessEvents pulls messages off the inboundEvents channel.
// Takes the events which were pulled off the bus, converts them to Event, and
// places onto the parsedEvents channel.
func ProcessEvents(inboundEvents chan []byte, parsedEvents chan *Event) {
	go func(inboundEvents chan []byte, parsedEvents chan *Event) {
		for event := range inboundEvents {
			var e Event
			json.Unmarshal(event, &e)
			parsedEvents <- &e
		}
	}(inboundEvents, parsedEvents)
}


func ProcessCommand(url string, body string, uniqueId string, method string) []byte {
	commandResult := make(chan []byte)
	inFlightCommands[uniqueId] = commandResult
	jsonMessage, err := json.Marshal(Command{UniqueID: uniqueId, URL: url, Method: method, Body: body})
	fmt.Println(jsonMessage)
	if err != nil {
		return []byte("")
	}

	commandChannel <- jsonMessage
	for {
		select {
		case r, r_ok := <- commandResult:
			if r_ok {
				return r
			}
		case <-time.After(5 * time.Second):
			return []byte("")
		}
	}
}
