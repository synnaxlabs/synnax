// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

// Handle is the materialization seam for a character insertion. Collaborative edits are
// relayed between clients without the server materializing them into the module's text,
// so it returns the state unchanged; durable materialization into Arc.Text is a
// follow-on concern (see SY-4393).
func (p InsertCharPayload) Handle(state Arc) (Arc, error) { return state, nil }

// Handle is the materialization seam for a character deletion. See InsertCharPayload.Handle.
func (p DeleteCharPayload) Handle(state Arc) (Arc, error) { return state, nil }
