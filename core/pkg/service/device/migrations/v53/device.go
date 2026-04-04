// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v53

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

type RackKey uint32

type Key = string

type StatusDetails struct {
	Rack   RackKey `json:"rack" msgpack:"rack"`
	Device string  `json:"device" msgpack:"device"`
}

type Device struct {
	Key        Key                           `json:"key" msgpack:"key"`
	Rack       RackKey                       `json:"rack" msgpack:"rack"`
	Location   string                        `json:"location" msgpack:"location"`
	Make       string                        `json:"make" msgpack:"make"`
	Model      string                        `json:"model" msgpack:"model"`
	Name       string                        `json:"name" msgpack:"name"`
	Configured bool                          `json:"configured" msgpack:"configured"`
	Properties msgpack.EncodedJSON           `json:"properties" msgpack:"properties"`
	Status     *status.Status[StatusDetails] `json:"status,omitempty" msgpack:"status,omitempty"`
	Parent     *ontology.ID                  `json:"parent,omitempty" msgpack:"parent,omitempty"`
}

var _ gorp.Entry[Key] = Device{}

func (d Device) GorpKey() Key    { return d.Key }
func (d Device) SetOptions() []any { return nil }

func OntologyID(key string) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeDevice, Key: key}
}
