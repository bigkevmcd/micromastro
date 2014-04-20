package main

// Handles notifications from Jenkins and queues a message in RabbitMQ with the
// details of the notification.

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/bitly/go-nsq"
)

var (
	nsqURI   = flag.String("uri", "localhost:4150", "NSQ URI")
	port     = flag.String("port", ":8080", "Listen on port")
	chanSize = flag.Int("queue", 5, "Size of channel for notifications")
	key      = flag.String("key", "notifications.jenkins.build", "Routing key")
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
	writer := nsq.NewWriter(*nsqURI)

	log.Printf("Connected to %s\n", *nsqURI)
	for {
		select {
		case n := <-notifications:
			body, err := json.Marshal(n)
			if err != nil {
				log.Println(err)
				continue
			}
			_, _, err = writer.Publish(*key, body)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
