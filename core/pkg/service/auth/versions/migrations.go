// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package versions holds the frozen shapes of the credentials table that preceded the
// password table, and the migrations that lift it to its last shape.
package versions

import (
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/x/types"
)

// Namespace is the applied-migration namespace of the credentials table.
var Namespace = types.Name[v1.SecureCredentials]()

// Migrations is the ordered migration chain that brings stored credentials to their
// last shape, v1, before the password table takes them over.
var Migrations = []gorp.Migration{v0.NormalizeKeys, v1.Migration}
