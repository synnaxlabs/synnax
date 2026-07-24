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

	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v2"
	"github.com/synnaxlabs/x/migrate"
)

// Migrations is the ordered migration chain for stored arcs.
var Migrations = slices.Concat(
	[]migrate.Migration{v0.Migration}, v1.Migrations, []migrate.Migration{v2.Migration},
)
