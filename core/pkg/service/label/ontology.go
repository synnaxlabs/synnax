// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/color"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// ResourceID constructs a unique resource.ID for the label with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeLabel, Key: k.String()}
}

// ResourceIDs constructs a slice of unique resource.IDs for the labels with the given
// keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":   zyn.UUID(),
	"name":  zyn.String(),
	"color": color.Schema,
})

func newResource(l Label) ontology.Resource {
	return ontology.NewResource(schema, l.ResourceID(), l.Name, l)
}

type change = xchange.Change[Key, Label]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() resource.Type { return resource.TypeLabel }

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
	var l Label
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&l).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(l), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ResourceID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(ctx context.Context, nexter iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Label]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func newSearchIndex() *gorp.SearchIndex[Key, Label] {
	return gorp.NewSearchIndex("search", func(e *Label) []string {
		return []string{e.Name}
	})
}
