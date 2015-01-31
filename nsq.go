package ari

import (
	"github.com/bitly/go-nsq"
	"github.com/bitly/nsq/util"
	"net/http"
	"encoding/json"
	"fmt"
	"bytes"
	"log"
	"errors"
)

var httpclient = &http.Client{}

// nsqConfig struct contains the NSQ specific configuration information.
type nsqConfig struct {
	Address				string   	`json:"nsq_address"`
	Channel				string		`json:"channel"`
	MaxInFlight			int			`json:"max_in_flight"`
	LookupdHttpAddress	[]string	`json:"lookupd_http_address"`
}

type NSQ struct {
	config nsqConfig
}

type topicInfo struct {
	StatusCode	int	`json:"status_code"`
}

func (n *NSQ) InitBus(config interface{}) error {
	n.config.MaxInFlight = 200
	c := config.(map[string]interface{})
	for key, value := range c {
		switch key {
		case "lookupd_http_address":
			lookupAddresses := value.([]interface{})
			for _, v := range lookupAddresses{
				n.config.LookupdHttpAddress = append(n.config.LookupdHttpAddress, v.(string))
			}
		case "channel":
			n.config.Channel = value.(string)
		case "max_in_flight":
			n.config.MaxInFlight = value.(int)
		case "nsq_address":
			n.config.Address = value.(string)
		}
	}
	
	// initial setup
	if len(n.config.LookupdHttpAddress) == 0 {
		return errors.New("Missing Lookupd HTTP Address configuration")
	}

	if n.config.Channel == "" {
		return errors.New("No application provided for connection to NSQ")
	}
	return nil
}

func (n *NSQ) StartProducer(topic string) (chan []byte, error) {
	messages := make(chan []byte)
	
		nsqcfg := nsq.NewConfig()
		nsqcfg.UserAgent = fmt.Sprintf("to_nsq/%s go-nsq/%s", util.BINARY_VERSION, nsq.VERSION)
		producer, err := nsq.NewProducer(n.config.Address, nsqcfg)
		if err != nil {
			return nil, err
		}
	go func(messages chan []byte, producer *nsq.Producer) {
		for message := range messages {
			producer.Publish(topic, message)
		}
	}(messages, producer)
	return messages, nil
}

func (n *NSQ) StartConsumer(topic string) (chan []byte, error) {
	// connect to nsq and get the json then save to a value
	cfg := nsq.NewConfig()
	cfg.UserAgent = fmt.Sprintf("go_ari_client/%s go-nsq/%s", util.BINARY_VERSION, nsq.VERSION)
	cfg.MaxInFlight = n.config.MaxInFlight

	// create new consumer and attach to lookupd
	consumer, err := nsq.NewConsumer(topic, n.config.Channel, cfg)
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
	err = consumer.ConnectToNSQLookupds(n.config.LookupdHttpAddress)
	if err != nil {
		return nil, err
	}
	
	return in, nil
}

func (n *NSQ) TopicExists(topic string) bool {
	var t topicInfo
	u := fmt.Sprintf("http://%s/lookup?topic=%s", n.config.LookupdHttpAddress[0], topic)
	fmt.Println(u)
	resp, err := httpclient.Get(u)
	if err != nil {
		fmt.Println(err)
		return false
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	err = json.Unmarshal(buf.Bytes(), &t)
	fmt.Println(t)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if t.StatusCode == 200 {
		return true
	} else {
		return false
	}
}