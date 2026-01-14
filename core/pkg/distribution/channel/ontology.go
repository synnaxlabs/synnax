// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"io"
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "channel"

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

func (c Channel) OntologyID() ontology.ID { return c.Key().OntologyID() }

// OntologyIDs returns the ontology.ID for each key.
func (k Keys) OntologyIDs() []ontology.ID {
	return lo.Map(k, func(key Key, _ int) ontology.ID { return key.OntologyID() })
}

func OntologyIDsFromChannels(chs []Channel) []ontology.ID {
	return lo.Map(chs, func(ch Channel, _ int) ontology.ID { return ch.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":         zyn.Uint32().Coerce(),
	"name":        zyn.String(),
	"leaseholder": zyn.Uint16().Coerce(),
	"is_index":    zyn.Bool(),
	"index":       zyn.Uint32().Coerce(),
	"data_type":   zyn.String(),
	"internal":    zyn.Bool(),
	"virtual":     zyn.Bool(),
	"expression":  zyn.String(),
})

func ToPayload(c Channel) map[string]any {
	return map[string]any{
		"key":         c.Key(),
		"name":        c.Name,
		"leaseholder": c.Leaseholder,
		"is_index":    c.IsIndex,
		"index":       c.Index(),
		"data_type":   c.DataType,
		"internal":    c.Internal,
		"virtual":     c.Virtual,
		"expression":  c.Expression,
		"operations":  c.Operations,
	}
}

func newResource(c Channel) ontology.Resource {
	return ontology.NewResource(schema, c.OntologyID(), c.Name, ToPayload(c))
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Channel]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k, err := ParseKey(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var ch Channel
	if err := s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(ch), nil
}

func translateChange(ch change) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     ch.Key.OntologyID(),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.NewObservable().OnChange(handleChange)
}

func (s *Service) NewObservable() observe.Observable[gorp.TxReader[Key, Channel]] {
	return gorp.Observe[Key, Channel](s.db)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[Key, Channel](s.db).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
