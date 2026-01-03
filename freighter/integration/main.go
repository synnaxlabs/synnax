// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	igrp "github.com/synnaxlabs/freighter/integration/grpc"
	"github.com/synnaxlabs/freighter/integration/http"
	xsig "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())
	http.BindTo(app)
	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)
	g := grpc.NewServer()
	s := igrp.New()
	s.BindTo(g)
	configureInstrumentation()

	err := func() error {
		sCtx, cancel := xsig.Isolated()
		sCtx.Go(func(ctx context.Context) error {
			return app.Listen(":8080")
		})

		lis, err := net.Listen("tcp", ":8081")
		if err != nil {
			return err
		}
		sCtx.Go(func(ctx context.Context) error {
			return g.Serve(lis)
		})
		<-interruptC
		g.Stop()
		if err := app.Shutdown(); err != nil {
			return err
		}
		cancel()
		return sCtx.Wait()
	}()

	if err != nil {
		zap.S().Fatalw("failed to start server", "error", err)
	}
}

func configureInstrumentation() {
	l, err := zap.NewDevelopmentConfig().Build()
	if err != nil {
		log.Fatal(err)
	}
	zap.ReplaceGlobals(l)
}
