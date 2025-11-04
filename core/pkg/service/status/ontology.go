// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

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

const OntologyType ontology.Type = "status"

// OntologyID returns the unique ID to identify the status within the Synnax ontology.
func OntologyID(k string) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k}
}

// OntologyIDs converts a slice of keys to a slice of ontology IDs.
func OntologyIDs(keys []string) (ids []ontology.ID) {
	return lo.Map(keys, func(k string, _ int) ontology.ID {
		return OntologyID(k)
	})
}

func KeyFromOntologyID(id ontology.ID) string {
	return id.Key
}

// KeysFromOntologyIDs converts a slice of ontology IDs to a slice of keys.
func KeysFromOntologyIDs(ids []ontology.ID) (keys []string) {
	keys = make([]string, len(ids))
	for i, id := range ids {
		keys[i] = KeyFromOntologyID(id)
	}
	return keys
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":         zyn.String(),
	"name":        zyn.String(),
	"variant":     zyn.String(),
	"message":     zyn.String(),
	"description": zyn.String(),
	"time":        zyn.Int64().Coerce(),
})

func newResource(s Status) ontology.Resource {
	return core.NewResource(schema, OntologyID(s.Key), s.Name, s)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[string, Status]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	var st Status
	err := s.NewRetrieve().WhereKeys(key).Entry(&st).Exec(ctx, tx)
	return newResource(st), err
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Status]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[string, Status](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[string, Status](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[Status, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
