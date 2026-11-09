// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command kafka runs one fake Kafka broker for Studio captures. It holds the topics of
// the docs examples and publishes oven readings to sensor-readings until it is stopped.
// Run it from the core module, which holds its dependencies.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	port := flag.Int("port", 9092, "the port the broker listens on")
	flag.Parse()
	cluster, err := kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.Ports(*port),
		kfake.SeedTopics(1, "sensor-readings", "synnax-samples"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Close()
	client, err := kgo.NewClient(kgo.SeedBrokers(cluster.ListenAddrs()...))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			elapsed := now.Sub(start).Seconds()
			value, err := json.Marshal(map[string]float64{
				"temperature": 180 + 5*math.Sin(elapsed/4),
				"pressure":    14 + math.Cos(elapsed/3),
			})
			if err != nil {
				log.Fatal(err)
			}
			client.Produce(
				ctx,
				&kgo.Record{
					Topic: "sensor-readings",
					Key:   []byte("oven"),
					Value: value,
				},
				func(_ *kgo.Record, err error) {
					if err != nil {
						log.Print(err)
					}
				},
			)
		}
	}
}
