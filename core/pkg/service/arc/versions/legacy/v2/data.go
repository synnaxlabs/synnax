// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v2 holds the frozen wire format for Console arc state at version 2.0.0.
// The wire model is structurally identical to v1; v2 marks the removal of the
// deprecated set_status STL symbol, whose props Migrate renames to status.set.
package v2

import (
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v1"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

// Version is the ImEx schema version of arc state at this version. The Console
// stamped it on the wire as the semver string "2.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 2

// Data is the wire shape of a Console arc state at version 2.0.0, structurally
// identical to v1.
type Data = v1.Data
