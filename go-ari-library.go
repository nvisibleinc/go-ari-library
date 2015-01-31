package ari

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var (
	inCommand chan *Event
)

type Event struct {
	ServerID  string    `json:"server_id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	ARI_Body  string    `json:"ari_body"`
}

type Command struct {
	UniqueID string `json:"unique_id"`
	URL      string `json:"url"`
	Body     string `json:"body"`
}

func UUID() string {
	f, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

func InitProducer(busType string, config interface{}, app string) chan []byte {
	var producer chan []byte
	switch busType {
	case "NSQ":
		producer = startNSQProducer(config, app)
	}
	return producer
}

// takes the events which were pulled off the bus, converts them to Event, and places onto the parsedEvents channel
func InitConsumer(inboundEvents chan []byte, parsedEvents chan *Event) {
	go func(inboundEvents chan []byte, parsedEvents chan *Event) {
		for event := range inboundEvents {
			var e Event
			json.Unmarshal(event, &e)
			parsedEvents <- &e
		}
	}(inboundEvents, parsedEvents)
}

// takes commands off the inCommand channel, convert to json, and place onto the outCommand channel as json
func InitProducer(outCommand chan []byte) {
	inCommand := make(chan *Command)
	go func(inCommand chan *Command, outCommand chan []byte) {
		for command := range inCommand {
			c, err := json.Marshal(command)
			if err != nil {
				fmt.Println(err)
			}
			outCommand <- c
		}
	}(inCommand, outCommand)
}
