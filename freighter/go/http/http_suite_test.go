// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = ShouldNotLeakGoroutinesPerSpec()

// newFiberApp builds a fiber app whose fasthttp worker-pool janitor wakes every
// 100ms instead of the default 10s. The janitor only observes shutdown between
// sleeps, so without this it can outlive the server long enough to trip the
// goroutine leak check.
func newFiberApp(cfg fiber.Config) *fiber.App {
	app := fiber.New(cfg)
	app.Server().MaxIdleWorkerDuration = 100 * time.Millisecond
	return app
}

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Suite")
}

func pollHealth(url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	return res.Body.Close()
}
