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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
	"github.com/synnaxlabs/x/gorp"
)

// String returns the key formatted as its decimal integer value.
func (k Key) String() string { return strconv.FormatUint(uint64(k), 10) }

// Rack returns the key of the rack this task belongs to.
func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

// LocalKey returns the task's unique key within its rack.
func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

// IsValid returns true when both the rack and local components are set.
func (k Key) IsValid() bool { return !k.Rack().IsZero() && k.LocalKey() != 0 }

// OntologyID returns the unique ontology identifier for the task.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTask, Key: k.String()}
}

var _ gorp.Entry[Key] = Task{}

// GorpKey implements gorp.Entry.
func (t Task) GorpKey() Key { return t.Key }

// SetOptions implements gorp.Entry.
func (Task) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the task.
func (t Task) OntologyID() ontology.ID { return t.Key.OntologyID() }
