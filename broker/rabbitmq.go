package broker

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	amqp "github.com/greatfocus/gf-amqp"
)

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
func Publish(queueName, routingKey, messageId string, data []byte) error {
	// get connection
	var appId = os.Getenv("APP_ID")
	expiry, err := strconv.ParseUint(os.Getenv("MESSAGE_EXPIRY"), 0, 64)
	expire := time.Duration(expiry) * time.Hour
	if err != nil {
		return err
	}
	conn, err := GetConn()
	if err != nil {
		return err
	}

	conn.err = conn.ch.ExchangeDeclare(
		appId,   // name
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if conn.err != nil {
		return conn.err
	}

	// create the queue if it doesn't already exist
	_, err = conn.ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// bind the queue to the routing key
	err = conn.ch.QueueBind(queueName, routingKey, appId, false, nil)
	if err != nil {
		return err
	}

	log.Printf("Broker published a message: %s", data)
	err = conn.ch.Publish(
		appId,      // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			AppId:        appId,
			MessageId:    messageId,
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			Expiration:   fmt.Sprint(expire),
		})
	if err != nil {
		return err
	}

	return nil
}

// StartConsumer -
func Consumer(queueName, routingKey string, handler func(d amqp.Delivery) bool, concurrency int) error {
	// get connection
	var appId = os.Getenv("APP_ID")
	conn, err := GetConn()
	if err != nil {
		return err
	}

	// prefetch 10x as many messages as we can handle at once
	prefetchCount := concurrency * 10
	err = conn.ch.Qos(prefetchCount, 0, false)
	if err != nil {
		return err
	}

	msgs, err := conn.ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
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
		go func() {
			for msg := range msgs {
				// if tha handler returns true then ACK, else NACK
				// the message back into the rabbit queue for
				// another round of processing
				if msg.AppId == appId && msg.RoutingKey == routingKey {
					log.Printf("Broker consumed a message: %s", msg.Body)
					if handler(msg) {
						_ = msg.Ack(false)
					} else {
						_ = msg.Nack(false, true)
					}
				}
				_ = msg.Nack(false, true)
			}
		}()
	}

	return nil
}
