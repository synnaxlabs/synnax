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
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
)

// V0Migrations bring stored users to their v0 shape.
var V0Migrations = []gorp.Migration{v0.NormalizeKeys, v0.Migration}

// Migrations is the ordered migration chain for stored users.
var Migrations = slices.Concat(V0Migrations, []gorp.Migration{v1.Migration})
