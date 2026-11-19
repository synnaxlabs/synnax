// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command sift runs the fake Sift server of the Core specs for Studio captures, with
// the assets of the docs examples. Run it from the core module, which holds its
// dependencies.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
)

func main() {
	port := flag.Int("port", 50551, "the port the server listens on")
	flag.Parse()
	srv, err := fake.Open(fake.Config{
		Address: fmt.Sprintf("127.0.0.1:%d", *port),
		Assets:  []string{"TPC", "Engine stand 2"},
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	<-ctx.Done()
	if err := srv.Close(); err != nil {
		log.Fatal(err)
	}
}
