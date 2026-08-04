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
	"log"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/synnaxlabs/freighter/integration/http"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

func main() {
	app := fiber.New(fiber.Config{})
	app.Use(logger.New())
	if err := http.BindTo(app); err != nil {
		log.Fatal(err)
	}
	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)
	configureInstrumentation()

	err := func() error {
		sCtx, cancel := xsignal.Isolated()
		sCtx.Go(func(context.Context) error {
			return app.Listen(":8080", fiber.ListenConfig{DisableStartupMessage: true})
		})
		<-interruptC
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
