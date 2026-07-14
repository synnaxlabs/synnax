// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v3 holds the frozen wire format for console line plot per-plot state
// at version 3.0.0. v3 moves the viewport mode, control state, and toolbar
// state from SliceState into the per-plot State. Those fields are UI-only and
// silently discarded on decode; the on-the-wire model is structurally
// identical to v2.
package v3

import v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v2"

const Version = "3.0.0"

// Data is the wire shape of a per-plot line plot state at v3.0.0. UI-only
// fields added at this version are ignored on decode.
type Data v2.Data
