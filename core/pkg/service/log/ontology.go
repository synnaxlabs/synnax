// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

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

// OntologyID returns unique identifier for the log within the ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeLog, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the logs within the ontology.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// OntologyIDsFromLogs returns the ontology IDs of the logs.
func OntologyIDsFromLogs(logs []Log) []ontology.ID {
	return lo.Map(logs, func(l Log, _ int) ontology.ID { return l.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID(), "name": zyn.String()})

func newResource(l Log) ontology.Resource {
	return ontology.NewResource(schema, l.OntologyID(), l.Name, l)
}

type change = xchange.Change[Key, Log]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeLog }

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
	var log Log
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&log).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(log), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Log]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, term, limit, OntologyID)
}
