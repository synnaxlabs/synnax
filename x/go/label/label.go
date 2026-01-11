// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ gorp.Entry[uuid.UUID] = Label{}

// GorpKey implements gorp.Entry.
func (l Label) GorpKey() uuid.UUID { return l.Key }

// SetOptions implements gorp.Entry.
func (l Label) SetOptions() []any { return nil }
