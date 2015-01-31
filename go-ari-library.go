package ari

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"log"
	"strings"
)

// Application struct contains the channels necessary
// for communication to/from the various message bus
// topics and the event channel
type AppInstance struct {
	inFlightCommands	map[string]chan *CommandResponse
	commandChannel		chan []byte
	responseChannel		chan []byte
	Events				chan *Event
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

// Command Response struct contains the response to a Command
type CommandResponse struct {
	UniqueID		string	`json:"unique_id"`
	StatusCode		int		`json:"status_code"`
	ResponseBody	string	`json:"response_body"`
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


func NewAppInstance() *AppInstance {
	var a AppInstance
	return &a
}
// InitAppInstance initializes the set of resources necessary
// for a new application
func (a *AppInstance) InitAppInstance(app string, busType string, config interface{}) {
	a.inFlightCommands = make(map[string]chan *CommandResponse)
	a.Events = make(chan *Event)
	commandTopic := strings.Join([]string{app, "commands"}, "_")
	responseTopic := strings.Join([]string{app, "responses"}, "_")
	a.commandChannel = InitProducer(commandTopic, busType, config)
	processEvents(InitConsumer(app, busType, config), a.Events)
	a.processCommandResponses(InitConsumer(responseTopic, busType, config))
}

func (a *AppInstance) processCommandResponses(inboundResponses chan []byte) {
		go func(inboundResponses chan []byte) {
		for response := range inboundResponses {
			var cr CommandResponse
			json.Unmarshal(response, &cr)
			returnChan, ok := a.inFlightCommands[cr.UniqueID]
			if ok {
				returnChan <- &cr
			}
			a.delInFlightCommand(cr.UniqueID)
		}
	}(inboundResponses)
}

// InitProducer initializes a new message bus producer.
// The InitProducer uses the configuration to determine which message bus to
// connect to, and is thus message bus agnostic for the proxy and client.
func InitProducer(app string, busType string, config interface{}) chan []byte {
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


func InitConsumer(app string, busType string, config interface{}) chan []byte {
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


func (a *AppInstance) delInFlightCommand(key string) {
	//TODO Add locking around this
	delete(a.inFlightCommands, key)
}

func (a *AppInstance) addInFlightCommand(key string, commandChan chan *CommandResponse) {
	//TODO Add locking around this
	a.inFlightCommands[key] = commandChan
}

func (a *AppInstance) processCommand(url string, body string, uniqueId string, method string) *CommandResponse {
	commandResponse := make(chan *CommandResponse)
	a.addInFlightCommand(uniqueId, commandResponse)
	jsonMessage, err := json.Marshal(Command{UniqueID: uniqueId, URL: url, Method: method, Body: body})
	if err != nil {
		return &CommandResponse{}
	}

	a.commandChannel <- jsonMessage
	for {
		select {
		case r, r_ok := <- commandResponse:
			if r_ok {
				return r
			}
		case <-time.After(5 * time.Second):
			return &CommandResponse{}
		}
	}
}
