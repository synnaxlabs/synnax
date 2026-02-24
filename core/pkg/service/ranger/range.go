// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ranger implements a service for managing ranges in a Synnax cluster. A range
// is a user defined region of time in a Synnax cluster. They act as a method for
// labeling and categorizing data.
package ranger

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Range (short for time range) is an interesting, user defined region of time in a
// Synnax cluster. They act as a method for labeling and categorizing data.
type Range struct {
	tx    gorp.Tx
	otg   *ontology.Ontology
	label *label.Service
	// Name is a human-readable name for the range. This name does not need to be
	// unique.
	Name string `json:"name" msgpack:"name"`
	// Color is the color used to represent the range in the UI.
	Color color.Color `json:"color" msgpack:"color"`
	// TimeRange is the range of time occupied by the range.
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
	// Key is a unique identifier for the Range. If not provided on creation, a new one
	// will be generated.
	Key uuid.UUID `json:"key" msgpack:"key"`
}

var _ gorp.Entry[uuid.UUID] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []any { return nil }

// UseTx binds a transaction to use for all query operations on the Range.
func (r Range) UseTx(tx gorp.Tx) Range { r.tx = tx; return r }

func (r Range) setOntology(otg *ontology.Ontology) Range { r.otg = otg; return r }

func (r Range) setLabel(label *label.Service) Range { r.label = label; return r }

// RetrieveParent returns the parent of the given range.
func (r Range) RetrieveParent(ctx context.Context) (Range, error) {
	var resources []ontology.Resource
	if err := r.otg.NewRetrieve().
		WhereIDs(r.OntologyID()).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(OntologyType).
		ExcludeFieldData(true).
		Entries(&resources).
		Exec(ctx, r.tx); err != nil {
		return Range{}, err
	}
	if len(resources) == 0 {
		return Range{}, errors.Wrapf(query.ErrNotFound, "range %s has no parent", r.Key)
	}
	key, err := KeyFromOntologyID(resources[0].ID)
	if err != nil {
		return Range{}, err
	}
	var res Range
	if err = gorp.NewRetrieve[uuid.UUID, Range]().
		WhereKeys(key).
		Entry(&res).
		Exec(ctx, r.tx); err != nil {
		return Range{}, err
	}
	res.tx = r.tx
	res.otg = r.otg
	return res.UseTx(r.tx), nil
}

func (r Range) RetrieveLabels(ctx context.Context) ([]label.Label, error) {
	return r.label.RetrieveFor(ctx, OntologyID(r.Key), r.tx)
}

// OntologyID returns the semantic ID for this range to look it up from within the
// ontology.
func (r Range) OntologyID() ontology.ID { return OntologyID(r.Key) }
