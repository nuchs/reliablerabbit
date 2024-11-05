package main

import (
	"context"
	"durex/internal/rampant"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	upQ   = "up"
	downQ = "down"
)

func main() {
	log.Printf("Hello! It's a me, right-io")
	declareQueues()

	client := rampant.NewClient(
		context.Background(),
		"amqp://guest:guest@michaelRabbit:5672",
		downQ,
		upQ,
	)
	defer client.Close()

	mainLoop(client)

	log.Printf("Ok lady! I love you buhbye")
}

func mainLoop(client *rampant.Client) {
	log.Printf("Starting main loop")
	ticker := time.NewTicker(time.Second * 3)
	count := 0
	recv := client.Recv()

	for {
		select {
		case <-ticker.C:
			count++
			data := fmt.Sprintf("Tick %d", count)
			client.Send(data)
		case msg := <-recv:
			log.Printf("Received a message %s", msg.Body)
		}
	}
}

func declareQueues() {
	conn, err := amqp.Dial("amqp://guest:guest@michaelRabbit:5672")
	if err != nil {
		log.Panicf("Shit! Failed to open connection: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Shit! Failed to open channel: %v", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(upQ, true, false, false, false, nil)
	if err != nil {
		log.Panicf("Shit! Failed to create upQ: %v", err)
	}

	_, err = ch.QueueDeclare(downQ, true, false, false, false, nil)
	if err != nil {
		log.Panicf("Shit! Failed to create downQ: %v", err)
	}
}
