package main


// Trivial message handler that just logs the messages received over NSQ.
// This should use ConnectToLookupd to discover producers for the topic.

import (
  "encoding/json"
  "flag"
  "log"
  "net/http"

  "github.com/bitly/go-nsq"
)

var (
  nsqURI   = flag.String("uri", "localhost:4150", "NSQ URI")
  key      = flag.String("key", "notifications.jenkins.build", "Routing key")
  chanName     = flag.String("channel", "jenkins.notifications.processor", "NSQ Channel name")
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

type MessageHandler struct {}

func  (m MessageHandler) HandleMessage(nm *nsq.Message) error {
  var not Notification
  err := json.Unmarshal(nm.Body, &not)
  if err != nil {
    return err
  }
  log.Println(not)
  return nil
}

func main() {
    n, err := nsq.NewReader(*key, *chanName)
    if err != nil {
      log.Fatal(err)
    }
    n.AddHandler(MessageHandler{})
    n.ConnectToNSQ(*nsqURI)

  http.Handle("/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
  http.ListenAndServe(":9192", nil)
}