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
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
)

// Migrations is the ordered migration chain for stored line plots.
var Migrations = []gorp.Migration{
	v0.NormalizeKeys,
	v0.Migration,
	v5.Migration,
	v6.Migration,
	v7.Migration,
	v8.Migration,
	v8.StrokeWidthMigration,
	v8.MembersMigration,
	v8.OwnersMigration,
	v9.RangeMigration,
	v9.Migration,
	v10.Migration,
}
