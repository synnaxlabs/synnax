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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v0"
	"github.com/synnaxlabs/x/gorp"
)

// Status is rack-specific status information including operational state.
type Status = status.Status[StatusDetails]

// StatusDetails is the rack-specific detail payload carried in a rack Status.
type StatusDetails struct {
	// Rack is the key of the rack the status describes.
	Rack Key `json:"rack" msgpack:"rack"`
}

// Rack is a manager for a collection of tasks deployed on a node.
type Rack struct {
	// Key uniquely identifies the rack in the cluster.
	Key Key `json:"key" msgpack:"key"`
	// Name is a human-readable name for the rack.
	Name string `json:"name" msgpack:"name"`
	// TaskCounter is the number of tasks ever created on the rack, used to issue
	// sequential local task keys.
	TaskCounter uint32 `json:"task_counter" msgpack:"task_counter"`
	// Embedded is true when the rack runs the node's embedded driver.
	Embedded bool `json:"embedded" msgpack:"embedded"`
	// Status is the last known status of the rack, if any.
	Status *Status `json:"status,omitempty" msgpack:"status,omitempty"`
	// Integrations are the driver integrations the rack supports.
	Integrations []string `json:"integrations" msgpack:"integrations"`
}

var _ gorp.Entry[Key] = Rack{}

// GorpKey implements gorp.Entry.
func (r Rack) GorpKey() Key { return r.Key }

// SetOptions implements gorp.Entry.
func (Rack) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the rack.
func (r Rack) OntologyID() ontology.ID { return r.Key.OntologyID() }
