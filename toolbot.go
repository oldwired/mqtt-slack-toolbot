package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/nlopes/slack"
)

var messagesChannel chan string
var postMessagesChannel chan string
var control chan bool

const (
	channelBufferSize          = 23
	messageChannelDepth        = 42
	toolbotOK           string = "1"
	toolbotNOK          string = "0"
)

type config struct {
	Secret         string   `json:"secret"`
	Channel        string   `json:"channel"`
	Topics         []string `json:"topics"`
	Topic4Channel  string   `json:"topic4channel"`
	BotStatusTopic string   `json:"botStatusTopic"`
	Debug          bool     `json:"debug"`
	Broker         string   `json:"broker"`
	Port           string   `json:"port"`
	ClientID       string   `json:"clientID"`
	EnterMessage   string   `json:"enterMessage"`
}

var handleSubscribedInboundMQTT mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	postMessagesChannel <- "TOPIC: " + msg.Topic() + " MSG: " + string(msg.Payload())
}

func checkSystemd() bool {
	dirInfo, err := os.Lstat("/run/systemd/system")
	if err != nil {
		return false
	}
	return dirInfo.IsDir()
}

func readConfig() *config {

	file, err := os.Open("config.json")
	if err != nil {
		panic(fmt.Sprintln("error:", err))
	}
	decoder := json.NewDecoder(file)
	result := config{}
	err = decoder.Decode(&result)
	if err != nil {
		panic(fmt.Sprintln("error:", err))
	}
	return &result
}

func sendMessagesFromChannel(api *slack.Client, config *config, params slack.PostMessageParameters) {
	for {
		api.PostMessage(config.Channel, <-postMessagesChannel, params)
	}
}

func doSlackAPI(config *config, wgReady *sync.WaitGroup) {
	//setup
	params := slack.PostMessageParameters{}
	params.AsUser = true

	api := slack.New(config.Secret)

	logger := log.New(os.Stdout, "toolbot ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	api.SetDebug(config.Debug)
	go sendMessagesFromChannel(api, config, params)

	//Bot is online
	wgReady.Done()
	postMessagesChannel <- config.EnterMessage

	//RTM
	go doSlackRTM(api, config)
}

func doSlackRTM(api *slack.Client, config *config) {
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for message := range rtm.IncomingEvents {
		switch event := message.Data.(type) {
		case *slack.MessageEvent:
			if config.Debug {
				fmt.Printf("Message: %v\n", event.Msg.Text)
			}
			users, _ := api.GetUsers() //to publish with the nick instead of ID -> TODO: quite expensive -> caching
			for _, user := range users {
				if user.ID == event.Msg.User {
					splitted := strings.Split(event.Msg.Timestamp, ".") //converting the time
					i, _ := strconv.ParseInt(splitted[0], 10, 64)
					t := time.Unix(i, 0)
					messagesChannel <- fmt.Sprint(t) + " " + user.Name + ": " + event.Msg.Text
				}
			}

		case *slack.RTMError:
			panic(fmt.Sprintf("Error: %s\n", event.Error()))

		case *slack.InvalidAuthEvent:
			panic(fmt.Sprintf("Invalid credentials"))

		default:
			// Ignore everything else
		}
	}
}

func doMQTT(config *config, wgReady *sync.WaitGroup) {
	if config.Debug {
		mqtt.DEBUG = log.New(os.Stdout, "", 0)
	}
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	mqttOptions := mqtt.NewClientOptions().AddBroker("tcp://" + config.Broker + ":" + config.Port).SetClientID(config.ClientID)
	mqttOptions.SetKeepAlive(2 * time.Second)
	mqttOptions.SetAutoReconnect(true)
	mqttOptions.SetMessageChannelDepth(messageChannelDepth)
	mqttOptions.SetWill(config.BotStatusTopic, toolbotNOK, 0, true)

	mqttClient := mqtt.NewClient(mqttOptions)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//Subscribe
	for _, topic := range config.Topics {
		if token := mqttClient.Subscribe(topic, 0, handleSubscribedInboundMQTT); token.Wait() && token.Error() != nil {
			panic(fmt.Sprintln(token.Error()))
		}
	}

	//Toolbot working
	wgReady.Done()
	token := mqttClient.Publish(config.BotStatusTopic, 0, true, toolbotOK)
	if !token.WaitTimeout(2000) {
		panic(fmt.Sprintln("Timeout waiting to publish"))
	}

	//sending the messages
	for {
		token := mqttClient.Publish(config.Topic4Channel, 0, false, <-messagesChannel)
		if !token.WaitTimeout(2000) {
			fmt.Println("Timeout waiting to publish")
		}
	}
}

func main() {
	messagesChannel = make(chan string, channelBufferSize)
	postMessagesChannel = make(chan string, channelBufferSize)
	control = make(chan bool)
	var wgReady sync.WaitGroup
	wgReady.Add(2) //2 Connectors: Slack and MQTT

	//Config
	config := readConfig()
	if config.Debug {
		fmt.Println(config.Secret)
	}

	//SLACK
	go doSlackAPI(config, &wgReady)

	//MQTT test
	go doMQTT(config, &wgReady)

	wgReady.Wait() //wait till both Connector-Routines report ready
	//Report Ready
	if checkSystemd() {
		daemon.SdNotify(false, "READY=1")
	} else {
		fmt.Println("Ready")
	}

	//block execution
	<-control
}
