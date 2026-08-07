// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v5 holds the frozen wire format for Console schematic state at version 5.
// v5 dropped the type literal and seeded the per-schematic mode and toolbar state.
// None are modeled here; the on-the-wire model is structurally identical to v4.
package v5

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 5

// Data is the persisted per-schematic state at version 5.
type Data v4.Data
