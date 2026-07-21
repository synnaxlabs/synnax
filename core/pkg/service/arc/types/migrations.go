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
	"slices"

	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v2"
)

// Migrations is the ordered migration chain for stored arcs.
var Migrations = append(slices.Clone(v1.Migrations), v2.Migration)
