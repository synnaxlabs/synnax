// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for Console schematic state at version 1.
// v1 attached a default control legend, which is UI-only and not modeled here; the
// on-the-wire model is structurally identical to v0.
package v1

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 1

// Data is the persisted per-schematic state at version 1.
type Data v0.Data
