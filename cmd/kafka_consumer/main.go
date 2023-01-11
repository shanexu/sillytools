package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"strings"
	"time"
)

type stringsFlags []string

func (i *stringsFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *stringsFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var brokers stringsFlags
	var groupID string
	var topic string
	flag.Var(&brokers, "brokers", "--brokers 127.0.0.1:9092")
	flag.StringVar(&groupID, "group", "sillytools_kafka_consumer", "--group xxxx")
	flag.StringVar(&topic, "topic", "", "--topic xxxtopic")
	flag.Parse()
	// make a new reader that consumes from topic-A, partition 0, at offset 42
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Println(time.Now().Sub(m.Time).Seconds())
	}

	if err := r.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}
}
