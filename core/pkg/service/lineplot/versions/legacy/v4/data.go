// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v4 holds the frozen wire format for Console line plot per-plot state at
// version 4.0.0. v4 adds measure mode and annotation visibility flags alongside the
// per-plot UI state introduced at v3. Both additions are UI-only and silently discarded
// on decode; the on-the-wire model is structurally identical to v3.
package v4

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v3 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v3"
)

// Version is the ImEx schema version of line plot data at this state. The Console
// stamped it on the wire as the semver string "4.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 4

// Data is the wire shape of a per-plot line plot state at v4.0.0. UI-only fields added
// at this version are ignored on decode.
type Data v3.Data
