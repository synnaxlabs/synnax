// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v4 holds the frozen wire format for Console schematic state at version 4.
// v4 added the default control authority, which is UI-only and not modeled here; the
// on-the-wire model is structurally identical to v3.
package v4

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 4

// Data is the persisted per-schematic state at version 4.
type Data v3.Data
