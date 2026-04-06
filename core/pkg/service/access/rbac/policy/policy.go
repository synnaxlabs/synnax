// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []any { return nil }
