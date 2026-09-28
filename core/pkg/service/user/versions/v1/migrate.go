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
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/gorp"
)

// Migration drops the username from stored users. It must run after the credential
// re-key, which copies each username onto the user's credentials.
var Migration = gorp.NewEntryMigration[Key, Key, v0.User, User](
	"v1_username_to_credentials",
	autoMigrateUser,
)
