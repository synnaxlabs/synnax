// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

// Change is one change that a repair makes to a task config, so that its links to the
// messages and fields of an ICD suit an edit of the ICD.
type Change struct {
	// Description says what changes, in plain words.
	Description string
	// Lossy is true when the change removes or alters a choice of the user. A config
	// with no lossy change takes the edit with no confirm.
	Lossy bool
}
