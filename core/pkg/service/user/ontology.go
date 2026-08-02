// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns a unique identifier for a User for use within a resource ontology.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeUser, Key: key.String()}
}

// OntologyIDsFromKeys returns a slice of unique identifiers from a slice of keys
func OntologyIDsFromKeys(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

// OntologyIDsFromUsers returns a slice of unique identifiers for a slice of Users for
// use within a resource ontology.
func OntologyIDsFromUsers(users []User) []ontology.ID {
	return lo.Map(users, func(u User, _ int) ontology.ID { return u.OntologyID() })
}

func KeyFromOntologyID(id ontology.ID) (Key, error) { return uuid.Parse(id.Key) }

var schema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"username":   zyn.String(),
	"first_name": zyn.String(),
	"last_name":  zyn.String(),
	"root_user":  zyn.Bool(),
})

var (
	_ ontology.Service      = (*Service)(nil)
	_ search.Service        = (*Service)(nil)
	_ search.FieldsProvider = (*Service)(nil)
)

func (*Service) Type() ontology.ResourceType { return ontology.ResourceTypeUser }

// SearchableFields implements ontology.SearchableFieldsProvider.
func (*Service) SearchableFields() []string {
	return []string{"username", "first_name", "last_name"}
}

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	uuidKey, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var u User
	if err = s.NewRetrieve().Entry(&u).Where(MatchKeys(uuidKey)).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(u), nil
}

type change = xchange.Change[Key, User]

func translateChange(ch change) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key).String(),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, User]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}

func newResource(u User) ontology.Resource {
	return ontology.NewResource(schema, u.OntologyID(), u.Username, u)
}
