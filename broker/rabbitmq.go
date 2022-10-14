package broker

import (
	"log"
	"os"

	amqp "github.com/greatfocus/gf-amqp"
)

var appId = "a6ced560-ea7a-4693-b990-7aada41982bf"

// Rabbitmq -
type Rabbitmq struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	err  error
}

// GetConn -
func GetConn() (*Rabbitmq, error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	rmq := &Rabbitmq{}

	rmq.conn, rmq.err = amqp.Dial(rabbitURL)
	if rmq.err != nil {
		return rmq, rmq.err
	}

	rmq.ch, rmq.err = rmq.conn.Channel()
	if rmq.err != nil {
		return rmq, rmq.err
	}

	return rmq, nil
}

// Publish -
func Publish(data []byte) error {
	// get connection
	conn, err := GetConn()
	if err != nil {
		return err
	}

	err = conn.ch.ExchangeDeclare(
		appId,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("Broker published a massage: %s", data)
	return conn.ch.Publish(
		// exchange - yours may be different
		appId,
		appId,
		// mandatory - we don't care if there I no queue
		false,
		// immediate - we don't care if there is no consumer on the queue
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
		})
}

// StartConsumer -
func Consumer(queueName string, handler func(d amqp.Delivery) bool, concurrency int) error {
	// get connection
	conn, err := GetConn()
	if err != nil {
		return err
	}

	err = conn.ch.ExchangeDeclare(
		appId,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// create the queue if it doesn't already exist
	_, err = conn.ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// bind the queue to the routing key
	err = conn.ch.QueueBind(queueName, appId, appId, false, nil)
	if err != nil {
		return err
	}

	// prefetch 4x as many messages as we can handle at once
	prefetchCount := concurrency * 4
	err = conn.ch.Qos(prefetchCount, 0, false)
	if err != nil {
		return err
	}

	msgs, err := conn.ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	// create a goroutine for the number of concurrent threads requested
	for i := 0; i < concurrency; i++ {
		log.Printf("Processing messages on thread %v...\n", i)
		go func() {
			for msg := range msgs {
				// if tha handler returns true then ACK, else NACK
				// the message back into the rabbit queue for
				// another round of processing
				log.Printf("Broker received a massage: %s", msg.Body)
				if handler(msg) {
					_ = msg.Ack(false)
				} else {
					_ = msg.Nack(false, true)
				}
			}
		}()
	}

	return nil
}
