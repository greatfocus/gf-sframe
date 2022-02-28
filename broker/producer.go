package broker

import (
	"log"
	"os"

	amqp "github.com/greatfocus/gf-amqp"
)

// Publish -
func Publish(exchange, exchangeType, queueName, routingKey string, data []byte, reliable bool) {
	URL := os.Getenv("RABBITMQ_URL")
	log.Printf("dialing %q", URL)
	connection, err := amqp.Dial(URL)
	if err != nil {
		log.Printf("Dial: %s", err)
	}
	defer func() {
		_ = connection.Close()
	}()

	log.Printf("got Connection, getting Channel")
	channel, err := connection.Channel()
	if err != nil {
		log.Printf("Channel: %s", err)
	}

	log.Printf("got Channel, declaring %q Exchange (%q)", exchangeType, exchange)
	if err = channel.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		log.Printf("Exchange Declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if reliable {
		log.Printf("enabling publishing confirms.")
		if err = channel.Confirm(false); err != nil {
			log.Printf("Channel could not be put into confirm mode: %s", err)
		}
		confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
		defer confirmOne(confirms)
	}

	log.Printf("declared Exchange, publishing %dB body (%q)", len(data), data)
	if err = channel.Publish(
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent, // 1=non-persistent, 2=persistent
			Priority:     0,               // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		log.Printf("Exchange Publish: %s", err)
	}
}

// One would typically keep a channel of publishings, a sequence number, and a
// set of unacknowledged sequence numbers and loop until the publishing channel
// is closed.
func confirmOne(confirms <-chan amqp.Confirmation) {
	log.Printf("waiting for confirmation of one publishing")

	if confirmed := <-confirms; confirmed.Ack {
		log.Printf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
	} else {
		log.Printf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
	}
}
