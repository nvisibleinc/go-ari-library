package ari

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"log"
	"strings"
)

// global variables
var bus MessageBus

// MessageBus interface
type MessageBus interface {
	InitBus(config interface{}) error
	StartProducer(topic string) (chan []byte, error)
	StartConsumer(topic string) (chan []byte, error)
	TopicExists(topic string) bool
}

// AppInstanceHandler
type AppInstanceHandler func(*AppInstance)

type App struct {
	name	string
	Events	chan []byte
	Stop	chan bool
}
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

func TopicExists(topic string) <-chan bool {
	c := make(chan bool)
	go func(topic string, c chan bool) {
		for i:= 0; i < 20; i++ {
			if bus.TopicExists(topic) {
			c <- true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}(topic, c)
	return c
}

func InitBus(busType string, config interface{}) error {
	switch busType {
	case "NSQ":
		// Start NSQ
		bus = new(NSQ)
	case "NATS":
		// Start NATS
		bus = new(NATS)
	case "OSLO":
		// Start an OSLO producer
		log.Fatal("OSLO message bus producer is not yet implemented.")
	case "RABBITMQ":
		// Start a RabbitMQ producer
		log.Fatal("RABBITMQ message bus producer is not yet implemented.")
	default:
		log.Fatal("No bus type was specified for the producer that we recognize.")
	}
	bus.InitBus(config)
	return nil
}

func NewApp() *App {
	var a App
	a.Stop = make(chan bool)
	return &a
}

func (a *App) Init(app string, handler AppInstanceHandler) {
	a.Events = InitConsumer(app)
	go func(app string, a *App) {
		for event := range a.Events {
			var as AppStart
			json.Unmarshal(event, &as)
			if as.Application == app {
				ai := new(AppInstance)
				ai.InitAppInstance(as.DialogID)
				go handler(ai)
			}
		}
	}(app, a)
}

// NewAppInstance function is a constructor to allocate the memory of
// AppInstance.
func NewAppInstance() *AppInstance {
	var a AppInstance
	return &a
}

// InitAppInstance initializes the set of resources necessary
// for a new application
func (a *AppInstance) InitAppInstance(instanceID string) {
	var err error
	a.Events = make(chan *Event)
	a.responseChannel = make(chan *CommandResponse)
	commandTopic := strings.Join([]string{"commands", instanceID}, "_")
	fmt.Println("Command topic is: ", commandTopic)
	responseTopic := strings.Join([]string{"responses", instanceID}, "_")
	a.commandChannel, err = bus.StartProducer(commandTopic)
	a.commandChannel <- []byte("DUMMY")
	if err != nil {
		fmt.Println(err)
	}
	eventBus, err :=  bus.StartConsumer(strings.Join([]string{"events", instanceID}, "_"))
	if err != nil {
		fmt.Println(err)
	}
	processEvents(eventBus, a.Events)
	responseBus, err := bus.StartConsumer(responseTopic)
	if err != nil {
		fmt.Println(err)
	}
	a.processCommandResponses(responseBus, a.responseChannel)
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
func InitProducer(topic string) chan []byte {
	producer, err := bus.StartProducer(topic)
	if err != nil {
		fmt.Println(err)
	}
	return producer
}

func InitConsumer(topic string) chan []byte {
	consumer, err := bus.StartConsumer(topic)
	if err !=nil {
		fmt.Println(err)
	}
	return consumer
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
