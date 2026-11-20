// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/gorp"

var _ gorp.Entry[Key] = Grant{}

// GorpKey implements gorp.Entry.
func (g Grant) GorpKey() Key { return g.Key }

// SetOptions implements gorp.Entry.
func (Grant) SetOptions() []any { return nil }
