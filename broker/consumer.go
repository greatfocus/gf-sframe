package broker

import (
	"fmt"
	"log"
	"os"
	"sync"

	amqp "github.com/greatfocus/gf-amqp"
)

// Consumer struct
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

// NewConsumer -
func NewConsumer(
	exchange,
	exchangeType,
	queueName,
	routingKey,
	ctag string,
	handler func(d amqp.Delivery, wg *sync.WaitGroup) bool) {
	var err error
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	URL := os.Getenv("RABBITMQ_URL")
	log.Printf("dialing %q", URL)
	c.conn, err = amqp.Dial(URL)
	if err != nil {
		log.Printf("Dial: %s", err)
	}

	go func() {
		log.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	log.Printf("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		log.Printf("Channel: %s", err)
	}

	// create the exchange if it doesn't already exist
	log.Printf("got Channel, declaring Exchange (%q)", exchange)
	if err = c.channel.ExchangeDeclare(exchange, exchangeType, true, false, false, false, nil); err != nil {
		log.Printf("Exchange Declare: %s", err)
	}

	// create the queue if it doesn't already exist
	log.Printf("declared Exchange, declaring Queue %q", queueName)
	queue, err := c.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Printf("Queue Declare: %s", err)
	}

	// bind the queue to the routing key
	log.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)", queue.Name, queue.Messages, queue.Consumers, routingKey)
	if err = c.channel.QueueBind(queue.Name, routingKey, exchange, false, nil); err != nil {
		log.Printf("Queue Bind: %s", err)
	}

	log.Printf("Queue bound to Exchange, starting Consume (consumer tag %q)", ctag)
	deliveries, err := c.channel.Consume(
		queue.Name, // queue
		ctag,       // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		log.Printf("Queue Consume: %s", err)
	}

	go handle(deliveries, c.done, handler)

	log.Printf("shutting down")
	if err := c.Shutdown(); err != nil {
		log.Fatalf("error during shutdown: %s", err)
	}
}

// Shutdown stop channel
func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	defer log.Printf("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

// handle the channel deliveries
func handle(deliveries <-chan amqp.Delivery, done chan error, handler func(d amqp.Delivery, wg *sync.WaitGroup) bool) {
	var wg sync.WaitGroup
	for d := range deliveries {
		log.Printf(
			"got %dB delivery: [%v] %q",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)
		// if tha handler returns true then ACK, else NACK
		// the message back into the rabbit queue for
		// another round of processing
		wg.Add(1)
		if handler(d, &wg) {
			_ = d.Ack(false)
		} else {
			_ = d.Nack(false, true)
		}
	}
	wg.Wait()
	log.Printf("handle: deliveries channel closed")
	done <- nil
}
