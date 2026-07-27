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
	v6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/migrate"
)

// Migrations is the ordered migration chain for stored schematics.
var Migrations = append([]migrate.Migration{v6.Migration}, v7.Migrations...)
