package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

/*
  structs for expose Kafka metrics
*/

type statReader struct{}

type statWriter struct{}

// structure for Alarm message
type kafkaMsg struct {
	Logsource   string `json:"logsource"`
	Logtype     string `json:"class"`
	Timestamp   string `json:"@timestamp"`
	ProjID      string `json:"type"`
	OrgID       string `json:"orgid"`
	Message     string `json:"message"`
	Summary     string `json:"summary"`
	Description string `json:"desc"`
}

// listener for Kafka reader metrics endpoint
func (s *statReader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	str, _ := json.Marshal(reader.Stats())
	w.Write([]byte(str))
}

//listener for Kafka writer metric endpoint
func (s *statWriter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	str, _ := json.Marshal(writer.Stats())
	w.Write([]byte(str))
}

// initialize Kafka reader
func GetKafkaReader(kafkaURL, topic, groupID string, logger *log.Logger) *kafka.Reader {
	brokers := strings.Split(kafkaURL, ",")
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
		QueueCapacity:  10000,
		//ReadBackoffMin: 1 * time.Millisecond,
		ErrorLogger: logger,
	})
}

//inintialize Kfka writer
func NewKafkaWriter(kafkaURL, topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		//Balancer: &kafka.LeastBytes{},
		ErrorLogger: logger,
		Async:       true,
	})
}

//send alarm to Kafka topic

func SendAlarm(message map[string]interface{}, regexp string, findings string, comment string) {

	msg := new(kafkaMsg)

	fields := []string{"logsource", "class", "type", "orgid", "message"}

	for _, f := range fields {

		s, found := message[f]

		if found {

			switch f {
			case "logsource":
				msg.Logsource = s.(string)

			case "class":
				msg.Logtype = s.(string)

			case "type":
				msg.ProjID = s.(string)

			case "orgid":
				msg.OrgID = s.(string)

			case "message":
				msg.Message = s.(string)

			}
		}

	}

	msg.Summary = "Rule " + regexp + " rised alarm " + comment
	msg.Description = "Found strings " + findings

	alrm, _ := json.Marshal(&msg)

	if *debug {
		logger.Print(string(alrm))
		return
	}

	str := kafka.Message{
		Key:   []byte("ti"), //[]byte(alert.BadIP)
		Value: alrm,
	}

	err := writer.WriteMessages(context.Background(), str)

	if err != nil {
		logger.Print(err)
	}

}
