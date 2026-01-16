// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyType is the type of a role within the ontology.
const OntologyType ontology.Type = "role"

// OntologyID returns the unique ontology.ID for the Role with the given key.
func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: key.String()}
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID(), "name": zyn.String()})

func newResource(r Role) ontology.Resource {
	return ontology.NewResource(schema, r.OntologyID(), r.Name, r)
}

// Type implements ontology.Service.
func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var r Role
	if err := s.NewRetrieve().WhereKeys(k).Entry(&r).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(r), nil
}

func translateChange(ch change.Change[uuid.UUID, Role]) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Role]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, Role](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, Role](s.cfg.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
