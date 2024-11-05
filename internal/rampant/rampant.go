package rampant

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const reconnectPeriod = 3 * time.Second

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	send   chan []byte
	recv   chan amqp.Delivery
}

func NewClient(ctx context.Context, addr, in, out string) *Client {
	log.Printf("Creating new rabbit client")
	ctx, cancel := context.WithCancel(ctx)
	client := Client{
		ctx,
		cancel,
		make(chan []byte),
		make(chan amqp.Delivery),
	}

	go client.connect("inbound", addr, rabbitReceive{
		ctx,
		in,
		client.recv,
	})
	go client.connect("inbound", addr, rabbitSend{
		ctx,
		in,
		client.send,
	})

	return &client
}

func (c Client) Close() {
	c.cancel()
	close(c.recv)
	log.Printf("Closed client")
}

func (c Client) Send(data string) {
	if c.ctx.Err() != nil {
		return
	}
	c.send <- []byte(data)
}

func (c Client) Recv() <-chan amqp.Delivery {
	return c.recv
}

func (c Client) connect(name, addr string, action actionable) {
	for {
		log.Printf("[%s] (re)connecting to %s", name, addr)
		conn, err := amqp.Dial(addr)
		if err != nil {
			log.Printf("[%s] Failed to connect to %s", name, addr)
			select {
			case <-c.ctx.Done():
				log.Printf("[%s] Client closed, exiting reconnection loop", name)
			case <-time.After(reconnectPeriod):
				continue
			}
		}

		log.Printf("[%s] Connected to %s", name, addr)
		err = c.open(conn, action)
		log.Printf("[%s] Connection closed", name)
		if err != nil {
			log.Printf("[%s] Due to %s", name, err)
		}
	}
}

func (c Client) open(conn *amqp.Connection, action actionable) error {
	connClosed := conn.NotifyClose(make(chan *amqp.Error))
	for {
		log.Printf("[%s] (re)opening channel", action.name())
		ch, err := conn.Channel()
		if err != nil {
			log.Printf("[%s] Failed to open channel", action.name())
			select {
			case <-c.ctx.Done():
				log.Printf("[%s] Client closed, exiting reopening loop", action.name())
			case <-time.After(reconnectPeriod):
				continue
			case err := <-connClosed:
				return err
			}
		}

		log.Printf("[%s] channel opend", action.name())
		err = action.run(ch)
		log.Printf("[%s] Channel closed", action.name())
		if err != nil {
			log.Printf("[%s] Due to %s", action.name(), err)
		}
	}
}

type actionable interface {
	run(*amqp.Channel) error
	name() string
}

type rabbitReceive struct {
	ctx  context.Context
	key  string
	dest chan<- amqp.Delivery
}

func (r rabbitReceive) name() string {
	return r.name()
}

func (r rabbitReceive) run(ch *amqp.Channel) error {
	chClosed := ch.NotifyClose(make(chan *amqp.Error))
	for {
		log.Printf("[%s] (re)starting consumer", r.key)
		consumer, err := ch.Consume(r.key, "", false, false, false, false, nil)
		if err != nil {
			log.Printf("[%s] Failed to start consumer %v", r.key, err)
			select {
			case <-r.ctx.Done():
				log.Printf("[%s] Client closed, exiting consumer loop", r.key)
			case <-time.After(reconnectPeriod):
				continue
			case err := <-chClosed:
				return err
			}
		}

		log.Printf("[%s] consumer started", r.key)
		r.reciverLoop(consumer)
		log.Printf("[%s] Consumer stopped", r.key)
	}
}

func (r *rabbitReceive) reciverLoop(consumer <-chan amqp.Delivery) {
	for {
		select {
		case <-r.ctx.Done():
			log.Printf("[%s] Client closed, exiting receiver loop", r.key)
		case msg, ok := <-consumer:
			if !ok {
				return
			}
			r.dest <- msg
		}
	}
}

type rabbitSend struct {
	ctx context.Context
	key string
	src <-chan []byte
}

func (s rabbitSend) name() string {
	return s.key
}

func (s rabbitSend) run(ch *amqp.Channel) error {
	chClosed := ch.NotifyClose(make(chan *amqp.Error))
	for {
		log.Printf("[%s] (re)starting consumer", s.key)
		select {
		case <-s.ctx.Done():
			log.Printf("[%s] Client closed, exiting send loop", s.key)
		case err := <-chClosed:
			return err
		case msg := <-s.src:
			err := ch.Publish("", s.key, false, false, amqp.Publishing{
				ContentType: "text/plain",
				Body:        msg,
			})
			if err != nil {
				log.Printf("Failed to publish message %q: %v", string(msg), err)
			}
		}
	}
}
