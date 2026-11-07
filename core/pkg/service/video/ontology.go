// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns the ontology identifier of the video view with key k.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeVideo, Key: k.String()}
}

// OntologyIDs returns the ontology identifiers of the video views with keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// OntologyIDsFromVideos returns the ontology identifiers of videos.
func OntologyIDsFromVideos(videos []Video) []ontology.ID {
	return lo.Map(videos, func(v Video, _ int) ontology.ID { return v.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID(), "name": zyn.String()})

func newResource(v Video) ontology.Resource {
	return ontology.NewResource(schema, v.OntologyID(), v.Name, v)
}

type change = xchange.Change[Key, Video]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeVideo }

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
	var v Video
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&v).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(v), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Video]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[ontology.ID],
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, OntologyID, q)
}
