// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
)

const (
	MIMEApplicationJSON    = fiber.MIMEApplicationJSON
	MIMEApplicationMsgPack = "application/msgpack"
)

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}
