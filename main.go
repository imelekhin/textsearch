package main

/*  Search for regexp patterns in Kafka stream. Kafka messages must be in JSON.
If regexp found write alarm to Kafka topic.
*/

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

/*
 Main hash map to store mapings from Kafka message field to array of regexp to search in that field
*/
type fieldsHashTable map[string]*searchList

/*
searchList stores compiled regexp, comments and index from regexp to comment. Multiply regexps can has single comment
*/

type searchList struct {
	regexps    []regexp.Regexp
	commentIdx []int
	comments   []string
}

/*
  structs for expose Kafka metrics
*/

type statReader struct{}

type statWriter struct{}

// structure for Alarm message
type kafkaMsg struct {
	SrcIP       string `json:"srcip"`
	Message     string `json:"message"`
	Summary     string `json:"summary"`
	Description string `json:"desc"`
	Timestamp   string `json:"@timestamp"`
	Type        string `json:"type"`
}

var (
	kafkaURL    = flag.String("kafka-broker", "127.0.0.1:9092", "Kafka broker URL list")
	intopic     = flag.String("kafka-in-topic", "notopic", "Kafka topic to read from")
	outtopic    = flag.String("kafka-out-topic", "notopic", "Kafka topic to write to")
	groupID     = flag.String("kafka-group", "nogroup", "Kafka group")
	metricsport = flag.String("metric-port", "1234", "Port to expose metrics")
	filename    = flag.String("config", "textsearch.cfg", "config file path name")
	fields      fieldsHashTable
	logger      *log.Logger
	reader      *kafka.Reader
	writer      *kafka.Writer
)

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
func getKafkaReader(kafkaURL, topic, groupID string, logger *log.Logger) *kafka.Reader {
	brokers := strings.Split(kafkaURL, ",")
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		//StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
		QueueCapacity:  10000,
		//ReadBackoffMin: 1 * time.Millisecond,
		ErrorLogger: logger,
	})
}

//inintialize Kfka writer
func newKafkaWriter(kafkaURL, topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		//Balancer: &kafka.LeastBytes{},
		ErrorLogger: logger,
		Async:       true,
	})
}

//service function for extration from line regexp or fielname and comment
func splitLine(line string, linenum int) ([]string, error) {
	res := []string{"", ""}
	str := strings.TrimSpace(line[1:])

	if string([]rune(str)[0]) != ":" {
		logger.Printf("Line %d: Error: no \":\" literal between command and regexp", linenum)
		return res, errors.New("")
	}
	str = strings.TrimSpace(str[1:])

	if string([]rune(str)[0]) != "'" {
		logger.Printf("Line %d: Error: no starting ' ", linenum)
		return res, errors.New("")
	}

	str1 := strings.Split(str[1:], "'")[0]
	str2 := strings.TrimSpace(strings.Split(str[1:], "'")[1])

	if string([]rune(str2)[0]) != ":" {
		logger.Printf("Line %d: Error: no \":\" literal between regexp and comment", linenum)
		return res, errors.New("")
	}
	str2 = strings.TrimSpace(str2[1:])
	res[0] = str1
	res[1] = str2
	return res, nil
}

//function for extract regexps from file for f: command

func loadRegexpFromFile(filename string) ([]regexp.Regexp, error) {

	var res []regexp.Regexp

	file, err := os.Open(filename)

	defer file.Close()

	if err != nil {
		logger.Print("Error opening file ", filename)
		return nil, errors.New("")
	}

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		logger.Print("Error opening file ", filename)
		return nil, errors.New("")
	}

	for scanner.Scan() {
		str := strings.TrimSpace(scanner.Text())
		r, err := regexp.Compile(str)
		if err != nil {
			logger.Print("Cannot compile rgexp ", str, " file ", filename)
			return nil, errors.New("")
		}

		res = append(res, *r)

	}

	return res, nil
}

// Load initial config file - mapping between fields and regexps
func loadSearches(filename string) {

	file, err := os.Open(filename)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	line := 1

	fieldset := false
	currentfield := ""

	for scanner.Scan() {
		currline := strings.TrimSpace(scanner.Text())

		if len(currline) <= 1 {
			line++

		} else {

			switch string([]rune(currline)[0]) {
			case "#":
				//logger.Printf("Line %d: this is comment line", line)
				line++
				break
			case "r":
				if !fieldset {
					logger.Printf("Line %d: error: no field set", line)
				} else {
					//logger.Printf("Line %d: this is single regexp", line)
					res, err := splitLine(currline, line)
					if err != nil {
						break
					}
					rgp, err := regexp.Compile(res[0])
					if err != nil {
						logger.Printf("Line %d : Cannot compile regexp", line)
						break
					} else {
						fields[currentfield].regexps = append(fields[currentfield].regexps, *rgp)
						fields[currentfield].comments = append(fields[currentfield].comments, res[1])
						fields[currentfield].commentIdx = append(fields[currentfield].commentIdx, len(fields[currentfield].comments)-1)
					}

				}
				line++
				break
			case "f":
				if !fieldset {
					logger.Printf("Line %d: error: no field set", line)

				} else {
					//logger.Printf("Line %d: this is file", line)
					res, err := splitLine(currline, line)
					if err != nil {
						line++
						break
					}

					fields[currentfield].comments = append(fields[currentfield].comments, res[1])

					rgp, err := loadRegexpFromFile(res[0])

					if err != nil {
						line++
						break
					}

					fields[currentfield].regexps = rgp

					for i := 0; i < len(fields[currentfield].regexps); i++ {
						fields[currentfield].commentIdx = append(fields[currentfield].commentIdx, len(fields[currentfield].comments)-1)
					}

				}
				line++
				break
			case "[":
				//logger.Printf("Line %d: this is start of field definition", line)
				fieldset = true
				currentfield = strings.TrimSpace(strings.Split(currline[1:], "]")[0])
				if len(currentfield) == 0 {
					logger.Printf("Line %d: Error: empty field", line)
					line++
					break
				}

				fields[currentfield] = new(searchList)

				line++
				break

			}
		}

	}

}

//send alarm to Kafka topic

func sendAlarm(message map[string]interface{}, regexp string, findings string, comment string) {

	msg := new(kafkaMsg)
	msg.Message = message["message"].(string)
	msg.Type = message["type"].(string)
	msg.SrcIP = message["srcip"].(string)
	msg.Summary = comment
	msg.Description = "Found string " + findings + "with regexp '" + regexp + "'"
	msg.Timestamp = time.Now().Format(time.RFC3339)

	logger.Print(msg)

	alrm, _ := json.Marshal(&msg)

	str := kafka.Message{
		Key:   []byte("ti"), //[]byte(alert.BadIP)
		Value: alrm,
	}

	err := writer.WriteMessages(context.Background(), str)

	if err != nil {
		logger.Print(err)
	}

}

func init() {
	logger = log.New(os.Stdout, "textsearch: ", log.Ldate|log.Ltime)
	fields = make(fieldsHashTable, 0)
	flag.Parse()

	logger.Print("Loading config from ", *filename)
	loadSearches(*filename)

	for k, j := range fields {
		logger.Print("Field is: ", k)
		for i := 0; i < (len(j.regexps)); i++ {
			logger.Print("On regexp '", j.regexps[i].String(), "' comment is ", j.comments[j.commentIdx[i]], " line ", i)
		}

	}
	logger.Print("Done loading config")

}

func main() {

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	reader = getKafkaReader(*kafkaURL, *intopic, *groupID, logger)
	writer = newKafkaWriter(*kafkaURL, *outtopic)

	

	defer func() {
		reader.Close()
		writer.Close()
		close(sigs)
	}()

	go func() {
		r := &statReader{}
		w := &statWriter{}
		http.Handle("/metrics/reader", r)
		http.Handle("/metrics/writer", w)
		logger.Fatal(http.ListenAndServe(":"+*metricsport, nil))
	}()

	logger.Print("start consuming ... !!")

	start := time.Now()
	msgcount := 0

	//  var readTime time.Duration = 0
	//	var searchTime time.Duration = 0
	//  var writeTime time.Duration = 0
loop:
	for {

		select {
		case sig := <-sigs:
			logger.Print(sig)
			break loop
		default:
			//startTime := time.Now()

			m, err := reader.ReadMessage(context.Background())
			//readTime = readTime + time.Since(startTime)

			msgcount++

			if err != nil {
				logger.Print(err)
				break
			}

			var f interface{}

			err = json.Unmarshal([]byte(m.Value), &f)

			if err != nil {
				logger.Print(err)
				break
			}

			msg := f.(map[string]interface{})

			for k, j := range msg {
				value, found := fields[k]
				if found {
					for i := 0; i < len((*value).regexps); i++ {
						str := (*value).regexps[i].FindString(j.(string))
						if len(str) > 0 {
							sendAlarm(msg, (*value).regexps[i].String(), str, (*value).comments[(*value).commentIdx[i]])
						}
					}
				}
			}
		}
	}

	logger.Print("Terminating")
	elapsed := time.Since(start)
	logger.Printf("Message processed %d in %s", msgcount, elapsed)
	//logger.Printf("Read time: %s Search time: %s Write time %s", readTime, searchTime, writeTime)
}
