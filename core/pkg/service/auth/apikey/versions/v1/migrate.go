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

	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/apikey/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// Migration lifts stored API keys from v0 to v1. A v0 key never expires.
var Migration = migrate.NewEntry(
	"v63_api_key_expires",
	func(ctx context.Context, old v0.APIKey) (APIKey, error) {
		k, err := autoMigrateAPIKey(ctx, old)
		if err != nil {
			return APIKey{}, err
		}
		k.Expires = telem.TimeStampMax
		return k, nil
	},
)
