// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command sift runs the fake Sift server of the Core specs for Studio captures, with
// the assets of the docs examples. The fake listens on a free port, so the command
// forwards a fixed port to it. Run it from the core module, which holds its
// dependencies.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
)

func main() {
	port := flag.Int("port", 50551, "the port the server listens on")
	flag.Parse()
	srv, err := fake.Open(fake.Config{Assets: []string{"TPC", "Engine stand 2"}})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = srv.Close() }()
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = lis.Close()
	}()
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go forward(conn, srv.Address())
	}
}

// forward copies bytes both ways between conn and the fake at address until either
// side closes.
func forward(conn net.Conn, address string) {
	defer func() { _ = conn.Close() }()
	upstream, err := net.Dial("tcp", address)
	if err != nil {
		log.Print(err)
		return
	}
	defer func() { _ = upstream.Close() }()
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go pipe(upstream, conn)
	go pipe(conn, upstream)
	<-done
}
