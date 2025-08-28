// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "device"

func OntologyID(k string) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k}
}

func OntologyIDsFromDevices(ds []Device) []ontology.ID {
	return lo.Map(ds, func(item Device, _ int) ontology.ID {
		return OntologyID(item.Key)
	})
}

func OntologyIDs(keys []string) []ontology.ID {
	return lo.Map(keys, func(key string, _ int) ontology.ID {
		return OntologyID(key)
	})
}

func KeysFromOntologyIDs(ids []ontology.ID) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.Key
	}
	return keys
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.String(),
	"name":       zyn.String(),
	"make":       zyn.String(),
	"model":      zyn.String(),
	"configured": zyn.Bool(),
	"location":   zyn.String(),
	"rack":       zyn.Uint32().Coerce(),
})

func newResource(r Device) ontology.Resource {
	return core.NewResource(schema, OntologyID(r.Key), r.Name, r)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[string, Device]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	var r Device
	err := s.NewRetrieve().WhereKeys(key).Entry(&r).Exec(ctx, tx)
	return newResource(r), err
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Device]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[string, Device](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[string, Device](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Device, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
