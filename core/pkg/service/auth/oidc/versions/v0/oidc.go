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

var _ gorp.Entry[Key] = Link{}

// GorpKey implements gorp.Entry.
func (l Link) GorpKey() Key { return l.Key }

// SetOptions implements gorp.Entry.
func (Link) SetOptions() []any { return nil }
