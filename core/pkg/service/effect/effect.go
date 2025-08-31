// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/status"
)

// Effect is a definition for a condition that results in a set of effects being
// executed.
type Effect struct {
	// tx is the transaction used to execute operations specific to this effect
	tx    gorp.Tx
	label *label.Service
	// Key is a unique identifier for the Effect. If not provided on creation, a new one
	// will be generated.
	Key     uuid.UUID     `json:"key" msgpack:"key"`
	Name    string        `json:"name" msgpack:"name"`
	Slate   uuid.UUID     `json:"slate" msgpack:"slate"`
	Enabled bool          `json:"enabled" msgpack:"enabled"`
	Labels  []label.Label `json:"labels" msgpack:"labels"`
	Status  *Status       `json:"status,omitempty" msgpack:"status,omitempty"`
}

var _ gorp.Entry[uuid.UUID] = Effect{}

// GorpKey implements gorp.Entry.
func (e Effect) GorpKey() uuid.UUID { return e.Key }

// SetOptions implements gorp.Entry.
func (e Effect) SetOptions() []any { return nil }

// UseTx binds a transaction to use for all query operations on the Effect.
func (e Effect) UseTx(tx gorp.Tx) Effect { e.tx = tx; return e }

// setLabel binds a label service to the Effect.
func (e Effect) setLabel(label *label.Service) Effect { e.label = label; return e }

// RetrieveLabels retrieves all labels associated with this effect.
func (e Effect) RetrieveLabels(ctx context.Context) ([]label.Label, error) {
	return e.label.RetrieveFor(ctx, OntologyID(e.Key), e.tx)
}

// OntologyID returns the semantic ID for this effect to look it up from within
// the ontology.
func (e Effect) OntologyID() ontology.ID { return OntologyID(e.Key) }

type StatusDetails struct {
	Effect uuid.UUID
}

type Status = status.Status[StatusDetails]
