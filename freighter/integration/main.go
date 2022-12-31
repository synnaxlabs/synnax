// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/synnaxlabs/freighter/integration/server"
	"go.uber.org/zap"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())
	server.BindTo(app, zap.S())
	if err := app.Listen(":8080"); err != nil {
		zap.S().Fatalw("server failed", "err", err)
	}
}
