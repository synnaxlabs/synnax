// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"io"
	"iter"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

// entry is the smallest resource shape, used only to assert the service's
// interfaces at compile time.
type entry struct {
	Key  uuid.UUID
	Rack rack.Key
	Name string
}

func (e entry) GorpKey() uuid.UUID { return e.Key }

func (e entry) SetOptions() []any { return nil }

var (
	_ ontology.Service      = (*Service[entry])(nil)
	_ search.Service        = (*Service[entry])(nil)
	_ search.FieldsProvider = (*Service[entry])(nil)
)

// Type implements ontology.Service.
func (s *Service[D]) Type() ontology.ResourceType { return s.cfg.Type }

// SearchableFields implements search.FieldsProvider.
func (s *Service[D]) SearchableFields() []string { return s.cfg.SearchableFields }

// RetrieveResource implements ontology.Service.
func (s *Service[D]) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var d D
	if err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, D](k)).
		Entry(&d).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		return ontology.Resource{}, err
	}
	return s.newResource(d), nil
}

// OnChange implements ontology.Service.
func (s *Service[D]) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	return s.table.Observe().OnChange(
		func(ctx context.Context, reader gorp.TxReader[uuid.UUID, D]) {
			f(ctx, xiter.Map(reader, s.translateChange))
		},
	)
}

// OpenNexter implements search.Service.
func (s *Service[D]) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, s.newResource), closer, nil
}

func (s *Service[D]) newResource(d D) ontology.Resource {
	return ontology.NewResource(
		s.cfg.Schema,
		s.OntologyID(d.GorpKey()),
		s.cfg.NameOf(d),
		d,
	)
}

func (s *Service[D]) translateChange(c change.Change[uuid.UUID, D]) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     s.OntologyID(c.Key).String(),
		Value:   s.newResource(c.Value),
	}
}
