// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = ConfigRecord{}

// GorpKey implements gorp.Entry.
func (c ConfigRecord) GorpKey() uuid.UUID { return c.Key }

// SetOptions implements gorp.Entry.
func (ConfigRecord) SetOptions() []any { return nil }

// SetKey sets the key of the stored configuration record.
func (c *ConfigRecord) SetKey(key uuid.UUID) { c.Key = key }
