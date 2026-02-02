// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = Table{}

// GorpKey implements gorp.Entry.
func (t Table) GorpKey() uuid.UUID { return t.Key }

// SetOptions implements gorp.Entry.
func (t Table) SetOptions() []any { return nil }
