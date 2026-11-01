// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
)

// Migrations is the ordered migration chain for stored arcs.
var Migrations = slices.Concat(
	[]gorp.Migration{v0.NormalizeKeys, v0.Migration},
	v1.Migrations,
	[]gorp.Migration{v3.Migration, v3.ModeMigration, v4.Migration},
)
