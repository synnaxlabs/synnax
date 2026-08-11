// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import "github.com/synnaxlabs/x/set"

// versionOwnedGo are the @go expressions a version file records: the
// persistence declarations, plus the imex marker that dates a resource's
// export history.
var versionOwnedGo = set.New("marshal", "hand", "migrate", "pinned", "imex")

// VersionOwned reports whether a domain expression belongs to the version
// files: the live file must match chain resolution on it exactly. Everything
// else (outputs, language bindings, validation, indexing) is live-owned and
// authored directly in the live file.
func VersionOwned(domain, exprName string) bool {
	switch domain {
	case "key", "doc":
		return true
	case "go":
		return versionOwnedGo.Contains(exprName)
	default:
		return false
	}
}
