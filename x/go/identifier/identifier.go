// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package identifier provides utilities for resolving entity references that
// may appear as either a UUID-form key or a human-readable name.
package identifier

import "github.com/google/uuid"

// IsKey reports whether s parses as a UUID-form key. Callers that accept a
// key-or-name input use it to dispatch between by-key and by-name lookups.
func IsKey(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
