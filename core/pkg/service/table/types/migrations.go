// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/types/v2"
	"github.com/synnaxlabs/x/migrate"
)

// Migrations returns the ordered migration chain for stored tables.
func Migrations() []migrate.Migration {
	return []migrate.Migration{v2.CodecMigration, v2.Migration}
}
