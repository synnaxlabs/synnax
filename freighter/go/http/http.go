// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
)

// BindableTransport is a freighter.Transport that knows how to register its routes on a
// fiber.App. The Router and any individual server registered through it satisfy this
// interface.
type BindableTransport interface {
	freighter.Transport
	// BindTo registers the transport's HTTP and websocket routes on the given
	// fiber.App.
	BindTo(*fiber.App)
}
