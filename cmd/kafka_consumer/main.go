package main

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

func main() {
	fmt.Println("hello world")
	// make a new reader that consumes from topic-A, partition 0, at offset 42
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"172.16.130.218:9092", "172.16.128.192:9092", "172.16.130.193:9092"},
		GroupID:  "sillytools_kafka_consumer",
		Topic:    "ads.fi.bond.inc.5100.mergebroker",
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
