package ari

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"log"
	"strings"
)

// AppInstance struct contains the channels necessary for communication to/from
// the various message bus topics and the event channel.
type AppInstance struct {
	commandChannel		chan []byte
	responseChannel		chan *CommandResponse
	quit				chan int
	Events				chan *Event
}

// Event struct contains the events we pull off the websocket connection.
type Event struct {
	ServerID  string    `json:"server_id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	ARI_Body  string    `json:"ari_body"`
}

type AppStart struct {
	Application	string	`json:"application"`
	DialogID	string	`json:"dialog_id"`
}

// Command struct contains the command we're passing back to ARI.
type Command struct {
	UniqueID	string	`json:"unique_id"`
	URL			string	`json:"url"`
	Method		string 	`json:"method"`
	Body		string	`json:"body"`
}

// CommandResponse struct contains the response to a Command
type CommandResponse struct {
	UniqueID		string	`json:"unique_id"`
	StatusCode		int		`json:"status_code"`
	ResponseBody	string	`json:"response_body"`
}

// UUID generates and returns a universally unique identifier.
// TODO(Brad): Replace this with an imported package.
func UUID() string {
	f, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

// NewAppInstance function is a constructor to allocate the memory of
// AppInstance.
func NewAppInstance() *AppInstance {
	var a AppInstance
	return &a
}

// InitAppInstance initializes the set of resources necessary
// for a new application
func (a *AppInstance) InitAppInstance(app string, instanceID string, busType string, config interface{}) {
	a.Events = make(chan *Event)
	a.responseChannel = make(chan *CommandResponse)
	commandTopic := strings.Join([]string{"commands", instanceID}, "_")
	responseTopic := strings.Join([]string{"responses", instanceID}, "_")
	a.commandChannel = InitProducer(commandTopic, busType, config)
	processEvents(InitConsumer(strings.Join([]string{"events", instanceID}, "_"), busType, config), a.Events)
	a.processCommandResponses(InitConsumer(responseTopic, busType, config), a.responseChannel)
}

// processCommandResponses is a function for parsing the Command-Response.
// processCommandResponse returns an anonymous go routine which will listen for
// information on the channel and process them as they arrive.
func (a *AppInstance) processCommandResponses(fromBus chan []byte, toAppInstance chan *CommandResponse) {
		go func(fromBus chan []byte, toAppInstance chan *CommandResponse) {
		for response := range fromBus {
			var cr CommandResponse
			json.Unmarshal(response, &cr)
			toAppInstance <- &cr
		}
	}(fromBus, toAppInstance)
}

// InitProducer initializes a new message bus producer.
// The InitProducer uses the configuration to determine which message bus to
// connect to, and is thus message bus agnostic for the proxy and client.
func InitProducer(topic string, busType string, config interface{}) chan []byte {
	var producer chan []byte
	switch busType {
	case "NSQ":
		// Start an NSQ producer
		producer = startNSQProducer(config, topic)
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


func InitConsumer(topic string, busType string, config interface{}) chan []byte {
	var busFeed chan []byte
	switch busType {
	case "NSQ":
		// Star NSQ Consumer
		busFeed = startNSQConsumer(config, topic)
	default:
		log.Fatal("No bus type was specified for the consumer that we recognize")
	}
	return busFeed
}

// ProcessEvents pulls messages off the inboundEvents channel.
// Takes the events which were pulled off the bus, converts them to Event, and
// places onto the parsedEvents channel.
func processEvents(inboundEvents chan []byte, parsedEvents chan *Event) {
	go func(inboundEvents chan []byte, parsedEvents chan *Event) {
		for event := range inboundEvents {
			var e Event
			json.Unmarshal(event, &e)
			parsedEvents <- &e
		}
	}(inboundEvents, parsedEvents)
}

func (a *AppInstance) processCommand(url string, body string, method string) *CommandResponse {
	jsonMessage, err := json.Marshal(Command{URL: url, Method: method, Body: body})
	if err != nil {
		return &CommandResponse{}
	}

	a.commandChannel <- jsonMessage
	for {
		select {
		case r, r_ok := <- a.responseChannel:
			if r_ok {
				return r
			}
		case <-time.After(5 * time.Second):
			return &CommandResponse{}
		}
	}
}
