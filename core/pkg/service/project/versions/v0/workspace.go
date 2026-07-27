// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// GorpKey implements gorp.Entry.
func (w Workspace) GorpKey() Key { return w.Key }

// SetOptions implements gorp.Entry.
func (Workspace) SetOptions() []any { return nil }
