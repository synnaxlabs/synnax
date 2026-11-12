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
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
)

// NewReadMigrations returns the ordered migration chain for stored HTTP read task
// configs. The last step reconciles each field with its channel among channels.
func NewReadMigrations(channels v5.Channels) []gorp.Migration {
	return []gorp.Migration{
		v3.Migration,
		v4.Migration,
		v5.NewReadMigration(channels),
	}
}

// WriteMigrations is the ordered migration chain for stored HTTP write task configs.
var WriteMigrations = []gorp.Migration{v5.WriteMigration}
