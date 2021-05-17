package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	rpio "github.com/stianeikeland/go-rpio"
)

var sessionStatus bool = true
var counter int = 0
var start = time.Now()
var TOPIC string = "PIR"

type pirStruct struct {
	PIR bool
}

func saveResultToFile(filename string, result string) {
	file, errOpen := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if errOpen != nil {
		log.Fatal(errOpen)
	}
	byteSlice := []byte(result)
	_, errWrite := file.Write(byteSlice)
	if errWrite != nil {
		log.Fatal(errWrite)
	}
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Message received")
}

func publish(client mqtt.Client) {
	if !sessionStatus {
		doneString := "{\"Done\": \"True\"}"
		client.Publish(TOPIC, 0, false, doneString)
		return
	} else {
		pirPin := rpio.Pin(17)
		pirPin.Input()
		readValue := pirPin.Read()
		var pirReading bool
		if int(readValue) == 1 {
			pirReading = true
		} else {
			pirReading = false
		}
		currentPIR := pirStruct{
			PIR: pirReading,
		}
		jsonPIR := currentPIR.structToJSON()
		client.Publish(TOPIC, 0, false, string(jsonPIR))
		return
	}
}

func (ps pirStruct) structToJSON() []byte {
	jsonReading, jsonErr := json.Marshal(ps)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	return jsonReading
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

var ADDRESS string
var PORT = 1883

func main() {

	// Save the IP address
	if len(os.Args) <= 1 {
		fmt.Println("IP address must be provided as a command line argument")
		os.Exit(1)
	}
	ADDRESS = os.Args[1]
	fmt.Println(ADDRESS)

	// Check that RPIO opened correctly
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// End program with ctrl-C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(0)
	}()

	// Creat MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", ADDRESS, PORT))
	opts.SetClientID("go_mqtt_client_pir")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Publish to topic
	numIterations := 10000
	for i := 0; i < numIterations; i++ {
		if i == numIterations-1 {
			sessionStatus = false
		}
		publish(client)
	}

	// Disconnect
	client.Disconnect(100)

	end := time.Now()
	duration := end.Sub(start).Seconds()
	resultString := fmt.Sprint("PIR runtime = ", duration, "\n")
	saveResultToFile("piResultsGo.txt", resultString)
	fmt.Println("PIR runtime = ", duration)
}
