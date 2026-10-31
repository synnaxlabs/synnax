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
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
)

// Migrations is the ordered migration chain for stored schematics.
var Migrations = []gorp.Migration{
	v0.NormalizeKeys,
	v0.Migration,
	v7.Migration,
	v8.Migration,
	v8.ScaleMigration,
	v9.Migration,
	v9.ScaleMigration,
	v10.Migration,
	v11.Migration,
}
