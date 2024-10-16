package main

import (
	"context"
	"durex/internal/rampant"
	"log"
	"strings"
)

const (
	upQ   = "up"
	downQ = "down"
)

func main() {
	log.Printf("Hello! It's a me, left-io")

	client := rampant.NewClient(
		context.Background(),
		"amqp://guest:guest@michaelRabbit:5672",
		upQ,
		downQ,
	)
	defer client.Close()

	log.Printf("Waiting for messages")
	for d := range client.Recv() {
		log.Printf("Received a message: %s", d.Body)
		data := "To" + strings.TrimPrefix(string(d.Body), "Ti")
		client.Send(data)
		d.Ack(false)
	}

	log.Printf("Ok lady! I love you buhbye")
}
