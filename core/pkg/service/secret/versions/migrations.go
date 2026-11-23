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
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	v1 "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v1"
)

// NewMigrations returns the ordered migration chain for stored secrets. Each file
// secret stored before v1 moves to rk.
func NewMigrations(rk rack.Key) []gorp.Migration {
	return []gorp.Migration{v1.NewMigration(rk)}
}
