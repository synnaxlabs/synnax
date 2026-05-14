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
	"log"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/synnaxlabs/freighter/integration/http"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

const defaultAddr = ":8080"

func main() {
	configureInstrumentation()
	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)
	defer signal.Stop(interruptC)
	if err := run(defaultAddr, interruptC); err != nil {
		zap.S().Fatalw("integration server error", "error", err)
	}
}

// run builds the fiber application, starts listening on addr, and blocks until
// either the listener exits on its own or shutdownC fires. Errors are wrapped
// with the phase in which they occurred (build / listen / shutdown / serve)
// so the caller can distinguish "never came up" from "crashed mid-flight" from
// "couldn't shut down cleanly."
func run(addr string, shutdownC <-chan os.Signal) error {
	app, err := newApp()
	if err != nil {
		return errors.Wrap(err, "build")
	}
	serveErrC := make(chan error, 1)
	go func() {
		serveErrC <- app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	select {
	case err := <-serveErrC:
		// Listener exited before shutdown was requested.
		return errors.Wrap(err, "listen")
	case <-shutdownC:
	}
	if err := app.Shutdown(); err != nil {
		return errors.Wrap(err, "shutdown")
	}
	// Collect the listener goroutine's final error, if any.
	if err := <-serveErrC; err != nil {
		return errors.Wrap(err, "serve")
	}
	return nil
}

// newApp constructs the fiber application with all integration test handlers
// bound. Separated from run so tests can mount the same handlers without
// taking a network listener.
func newApp() (*fiber.App, error) {
	app := fiber.New(fiber.Config{})
	app.Use(logger.New())
	if err := http.BindTo(app); err != nil {
		return nil, err
	}
	return app, nil
}

func configureInstrumentation() {
	l, err := zap.NewDevelopmentConfig().Build()
	if err != nil {
		log.Fatal(err)
	}
	zap.ReplaceGlobals(l)
}
