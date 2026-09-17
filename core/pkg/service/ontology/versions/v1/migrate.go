// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
)

// MigrateID lifts a v0 identifier to v1. The stored bytes are unchanged: v1 only widens
// the resource type set.
func MigrateID(ctx context.Context, old v0.ID) (ID, error) {
	return autoMigrateID(ctx, old)
}
