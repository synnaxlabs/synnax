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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v0 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
)

// NewMigrations returns the migration chain of the channel table on the Core with the
// host key host.
func NewMigrations(host node.Key) []gorp.Migration {
	return []gorp.Migration{
		v0.NormalizeKeys,
		v0.Migration,
		v1.Migration,
		v1.MembersMigration,
		v1.IndexGroupMigration,
		v1.NameMigration,
		v1.NameRuleMigration,
		v1.NewClashMigration(host),
		v1.CalculationTextMigration,
	}
}
