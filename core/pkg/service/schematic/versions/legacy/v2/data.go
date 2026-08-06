// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v2 holds the frozen wire format for Console schematic state at version 2.
// v2 added the per-schematic key, a type literal, and a viewport mode. All three are
// UI-only or Console-local and not modeled here; the on-the-wire model is structurally
// identical to v1.
package v2

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 2

// Data is the persisted per-schematic state at version 2.
type Data v1.Data
