// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Task{}

// GorpKey implements gorp.Entry.
func (t Task) GorpKey() Key { return t.Key }

// SetOptions implements gorp.Entry, leasing the task to its rack's node.
func (t Task) SetOptions() []any { return []any{t.Key.Rack().Node()} }

// OntologyID returns the unique ontology identifier for the task.
func (t Task) OntologyID() ontology.ID { return t.Key.OntologyID() }

// Rack returns the key of the rack the task belongs to.
func (t Task) Rack() rack.Key { return t.Key.Rack() }

// String returns the task's name and key, or just the key when unnamed.
func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}
