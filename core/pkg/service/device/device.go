// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package device implements types and services for managing physical pieces of hardware
// in Synnax. This includes creating, retrieving, and updating devices.
package device

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Device{}

// GorpKey gives a unique key for the device for use in gorp.
func (d Device) GorpKey() Key { return d.Key }

// SetOptions returns nil.
func (d Device) SetOptions() []any { return nil }

// OntologyID returns the unique ID for the device within the ontology.
func (d Device) OntologyID() ontology.ID { return OntologyID(d.Key) }
