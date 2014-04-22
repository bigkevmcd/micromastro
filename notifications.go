package main

// Handles notifications from Jenkins and queues a message in RabbitMQ with the
// details of the notification.

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bigkevmcd/micromastro/utils"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-tigertonic"
	"github.com/streadway/amqp"
)

var (
	amqpURI      = utils.GetEnvWithDefault("MASTRO_AMQP_URI", "amqp://guest:guest@localhost:5672/")
	exchangeName = utils.GetEnvWithDefault("MASTRO_EXCHANGE", "amqp.fanout")
	port         = utils.GetEnvWithDefault("MASTRO_NOTIFICATIONS_PORT", ":8080")
	chanSize     = utils.GetEnvWithDefault("MASTRO_NOTIFICATIONS_QUEUE_SIZE", "5")
	key          = utils.GetEnvWithDefault("MASTRO_NOTIFICATIONS_KEY", "notifications.jenkins.build")
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
	chanSize, err := strconv.Atoi(chanSize)
	if err != nil {
		log.Fatal("MASTRO_NOTIFICATIONS_QUEUE_SIZE must be a valid integer")
	}
	notifications := make(chan Notification, chanSize)
	http.Handle("/notifications", tigertonic.Timed(NotificationsHandler{notifications}, key, nil))
	go sendNotifications(notifications)

	go metrics.Log(metrics.DefaultRegistry, 60e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	log.Printf("Listening on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func sendNotifications(notifications chan Notification) {
	connection, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("Dial: %s", err)
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("Channel: %s", err)
	}
	err = channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)
	if err != nil {
		log.Fatalf("Exchange Declare: %s", err)
	}
	log.Printf("Connected to %s\n", amqpURI)

	for {
		select {
		case n := <-notifications:
			body, err := json.Marshal(n)
			if err != nil {
				log.Println(err)
				continue
			}
			channel.Publish(
				exchangeName,
				key,
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
