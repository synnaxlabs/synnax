// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "channel"

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs returns the ontology.ID for each key.
func (k Keys) OntologyIDs() []ontology.ID {
	return lo.Map(k, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

func OntologyIDsFromChannels(chs []Channel) []ontology.ID {
	return lo.Map(chs, func(item Channel, _ int) ontology.ID {
		return OntologyID(item.Key())
	})
}

var _schema = ontology.NewSchema(OntologyType, map[string]zyn.Z{
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

func newResource(c Channel) core.Resource {
	return core.NewResource(_schema, OntologyID(c.Key()), c.Name, map[string]any{
		"key":         c.Key(),
		"name":        c.Name,
		"leaseholder": c.Leaseholder,
		"is_index":    c.IsIndex,
		"index":       c.Index(),
		"data_type":   c.DataType,
		"internal":    c.Internal,
		"virtual":     c.Virtual,
		"expression":  c.Expression,
	})
}

var _ ontology.Service = (*service)(nil)

type change = changex.Change[Key, Channel]

// Schema implements ontology.Service.
func (s *service) Schema() *core.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := MustParseKey(key)
	var ch Channel
	err := s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx, tx)
	return newResource(ch), err
}

func translateChange(ch change) core.Change {
	return core.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *service) OnChange(f func(context.Context, iter.Nexter[core.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
		f(ctx, iter.NexterTranslator[change, core.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return s.NewObservable().OnChange(handleChange)
}

func (s *service) NewObservable() observe.Observable[gorp.TxReader[Key, Channel]] {
	return gorp.Observe[Key, Channel](s.DB)
}

// OpenNexter implements ontology.Service.
func (s *service) OpenNexter() (iter.NexterCloser[core.Resource], error) {
	n, err := gorp.WrapReader[Key, Channel](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Channel, core.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
