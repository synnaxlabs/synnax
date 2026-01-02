// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = LinePlot{}

// GorpKey implements gorp.Entry.
func (p LinePlot) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements gorp.Entry.
func (p LinePlot) SetOptions() []any { return nil }
