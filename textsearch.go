package main

/*  Search for regexp patterns in Kafka stream. Kafka messages must be in JSON.
If regexp found write alarm to Kafka topic.
*/

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type variables map[string][]string

type rule struct {
	name      string
	alarm     string
	condition Cond
	action    int           //reserved
	execTime  time.Duration //for speed profiling
	execCount int64
}

//type rules map[string]rule

var (
	kafkaURL    = flag.String("kafka-broker", "127.0.0.1:9092", "Kafka broker URL list")
	intopic     = flag.String("kafka-in-topic", "notopic", "Kafka topic to read from")
	outtopic    = flag.String("kafka-out-topic", "notopic", "Kafka topic to write to")
	groupID     = flag.String("kafka-group", "nogroup", "Kafka group")
	metricsport = flag.String("metric-port", "1234", "Port to expose metrics")
	filename    = flag.String("config", "textsearch.cfg", "config file path name")
	debug       = flag.Bool("debug", false, "force debug outpoot")
	logger      *log.Logger
	reader      *kafka.Reader
	writer      *kafka.Writer
	expr        variables
	rulelist    []rule
	rulecount   int
)

func init() {

	var err error

	logger = log.New(os.Stdout, "textsearch: ", log.Ldate|log.Ltime)

	flag.Parse()

	expr = make(variables)

	logger.Print("Loading config from ", *filename)

	err = Load(*filename, expr)

	if err != nil {
		log.Fatal(err)

	}

	rulecount = len(rulelist)

	logger.Print("Done loading config")

}

func main() {

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	reader = GetKafkaReader(*kafkaURL, *intopic, *groupID, logger)
	writer = NewKafkaWriter(*kafkaURL, *outtopic)

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

			for i := 0; i < rulecount; i++ {
				start := time.Now()
				res, str := rulelist[i].condition.Eval(msg)
				rulelist[i].execCount++
				rulelist[i].execTime += time.Since(start)
				if res {
					SendAlarm(msg, rulelist[i].name, strings.Join(str, " , "), rulelist[i].alarm)
				}
			}

		}
	}

	logger.Print("Terminating")
	elapsed := time.Since(start)
	logger.Printf("Message processed %d in %s", msgcount, elapsed)
	logger.Println(rulecount)
	for i := 0; i < rulecount; i++ {
		logger.Println("Rule ", rulelist[i].name, " average exec time ", int64(rulelist[i].execTime.Microseconds())/rulelist[i].execCount, " executed ", rulelist[i].execCount)
	}
	//logger.Printf("Read time: %s Search time: %s Write time %s", readTime, searchTime, writeTime)
}
