// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen pre-SY-4504 storage shape of a schematic symbol, in which
// the specification was persisted as an untyped map. It exists solely to drive the
// migration that lifts stored symbols into the current typed form.
package v0

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = Symbol{}

// GorpKey implements gorp.Entry.
func (s Symbol) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (Symbol) SetOptions() []any { return nil }
