package main

// Handles notifications from Jenkins and queues a message in RabbitMQ with the
// details of the notification.

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/streadway/amqp"
)

var (
	amqpURI      = flag.String("uri", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchangeName = flag.String("exchange", "amqp.fanout", "Durable AMQP exchange name")
	port         = flag.String("post", ":8080", "Listen on port")
	chanSize     = flag.Int("queue", 5, "Size of channel for notifications")
	key          = flag.String("key", "notifications.jenkins.build", "Routing key")
)

type Notification struct {
	Build struct {
		Number float64 `json:"number"`
		Phase  string  `json:"phase"`
		Url    string  `json:"url"`
	} `json:"build"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

func init() {
	flag.Parse()
}

type NotificationsHandler struct {
	notifications chan Notification
}

func (n NotificationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var not Notification
	err := decoder.Decode(&not)
	if err != nil {
		log.Println(err)
	} else {
		n.notifications <- not
	}
}

func main() {
	notifications := make(chan Notification, *chanSize)
	http.Handle("/notifications", NotificationsHandler{notifications})
	go sendNotifications(notifications)

    log.Printf("Listening on %s\n", *port)
	http.ListenAndServe(*port, nil)
}

func sendNotifications(notifications chan Notification) {
	connection, err := amqp.Dial(*amqpURI)
	if err != nil {
		log.Fatalf("Dial: %s", err)
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("Channel: %s", err)
	}
	err = channel.ExchangeDeclare(
		*exchangeName, // name
		"fanout",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // noWait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Exchange Declare: %s", err)
	}

    log.Printf("Connected to %s\n", *amqpURI)
	for {
		select {
		case n := <-notifications:
			body, err := json.Marshal(n)
			if err != nil {
				log.Println(err)
				continue
			}
			channel.Publish(
				*exchangeName,
				*key,
				false, // mandatory
				false, // immediate
				amqp.Publishing{
					Headers:         amqp.Table{},
					ContentType:     "application/json",
					ContentEncoding: "",
					Body:            body,
					DeliveryMode:    amqp.Transient,
					Priority:        0,
				},
			)
		}
	}
}
