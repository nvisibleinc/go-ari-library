package ari

import (
	"github.com/bitly/go-nsq"
	"github.com/bitly/nsq/util"
	"fmt"
	"log"
)



// NSQ_Config struct contains the NSQ specific configuration information.
type NSQ_Config struct {
	Address				string   	`json:"nsq_address"`
	Application			string		`json:applications`
	LookupdHttpAddress	[]string	`json:lookupd_http_address`
}

// startNSQProducer starts a new NSQ producer by returning a channel that
// messages should be put onto, thereby producing to the message bus.
func startNSQProducer(config interface{}, topic string) chan []byte {
	var address string
	n := config.(map[string]interface{})
	for key, value := range n {
		switch key {
		case "nsq_address":
			address = value.(string)
		}
	}

	messages := make(chan []byte)
	go func(address string, topic string) {
		nsqcfg := nsq.NewConfig()
		nsqcfg.UserAgent = fmt.Sprintf("to_nsq/%s go-nsq/%s", util.BINARY_VERSION, nsq.VERSION)
		producer, err := nsq.NewProducer(address, nsqcfg)
		if err != nil {
			log.Fatal(err)
		}
		for message := range messages {
			producer.Publish(topic, message)
		}
	}(address, topic)
	return messages
}

// startNSQConsumer start a new NSQ consumer by returning a channel that
// messages should be put onto, thereby consuming from the message bus.
func startNSQConsumer(config interface{}, topic string) chan []byte {
	var channel string
	var lookupdHttpAddress	[]string
	//var	application			string
	maxInFlight := 200

	n := config.(map[string]interface{})
	for key, value := range n {
		switch key {
		case "lookupd_http_address":
			lookupAddresses := value.([]interface{})
			for _, v := range lookupAddresses{
				lookupdHttpAddress = append(lookupdHttpAddress, v.(string))
			}
		case "channel":
			channel = value.(string)
		case "max_in_flight":
			maxInFlight = value.(int)
		}
	}
	
	// initial setup
	if len(lookupdHttpAddress) == 0 {
		log.Fatal("Missing Lookupd HTTP Address configuration")
	}

	if channel == "" {
		log.Fatal("No application provided for connection to NSQ")
	}

	// connect to nsq and get the json then save to a value
	cfg := nsq.NewConfig()
	cfg.UserAgent = fmt.Sprintf("go_ari_client/%s go-nsq/%s", util.BINARY_VERSION, nsq.VERSION)
	cfg.MaxInFlight = maxInFlight

	// create new consumer and attach to lookupd
	consumer, err := nsq.NewConsumer(topic, channel, cfg)
	if err != nil {
		log.Fatal(err)
	}

	in := make(chan []byte)
	handlerFunc := func(m *nsq.Message) error {
						in <- m.Body
						return nil
					}

	// process events as they come off the bus
	consumer.AddHandler(nsq.HandlerFunc(handlerFunc))
	err = consumer.ConnectToNSQLookupds(lookupdHttpAddress)
	if err != nil {
		log.Fatal(err)
	}
	
	return in
}