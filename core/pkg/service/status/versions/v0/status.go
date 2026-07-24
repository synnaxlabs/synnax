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
func (s Status[Details]) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (Status[Details]) SetOptions() []any { return nil }

// CustomTypeName implements types.CustomTypeName to match the Gorp key prefix stored
// statuses were written under.
func (Status[Details]) CustomTypeName() string { return "Status" }
