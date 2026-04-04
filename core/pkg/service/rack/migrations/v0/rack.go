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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

type Key uint32

func (k Key) String() string { return strconv.Itoa(int(k)) }

type StatusDetails struct {
	Rack Key `json:"rack" msgpack:"rack"`
}

type Rack struct {
	Key          Key                           `json:"key" msgpack:"key"`
	Name         string                        `json:"name" msgpack:"name"`
	TaskCounter  uint32                        `json:"task_counter" msgpack:"task_counter"`
	Embedded     bool                          `json:"embedded" msgpack:"embedded"`
	Status       *status.Status[StatusDetails] `json:"status,omitempty" msgpack:"status,omitempty"`
	Integrations []string                      `json:"integrations" msgpack:"integrations"`
}

var _ gorp.Entry[Key] = Rack{}

func (r Rack) GorpKey() Key      { return r.Key }
func (r Rack) SetOptions() []any { return nil }

func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRack, Key: k.String()}
}
