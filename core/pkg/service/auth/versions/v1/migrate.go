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

	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	"github.com/synnaxlabs/x/gorp"
)

// Migration re-encodes stored credentials from the untagged MessagePack v0 shape into
// the current Orc format. A plain codec migration cannot do this: the current type's
// snake_case MessagePack tags do not match the Go field names legacy rows were stored
// under.
var Migration = gorp.NewEntryMigration(
	"v1_orc_credentials",
	func(_ context.Context, old v0.SecureCredentials) (SecureCredentials, error) {
		return SecureCredentials{Username: old.Username, Password: old.Password}, nil
	},
)
