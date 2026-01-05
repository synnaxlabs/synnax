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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "user"

// OntologyID returns a unique identifier for a User for use within a resource ontology.
func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: key.String()}
}

// OntologyIDsFromKeys returns a slice of unique identifiers from a slice of keys
func OntologyIDsFromKeys(keys []uuid.UUID) []ontology.ID {
	ids := make([]ontology.ID, len(keys))
	for i, key := range keys {
		ids[i] = OntologyID(key)
	}
	return ids
}

// OntologyIDFromUser returns a unique identifier for a User for use within a resource
// ontology.
func OntologyIDFromUser(u *User) ontology.ID {
	return OntologyID(u.Key)
}

// OntologyIDsFromUsers returns a slice of unique identifiers for a slice of Users for
// use within a resource ontology.
func OntologyIDsFromUsers(users []User) []ontology.ID {
	ids := make([]ontology.ID, len(users))
	for i, u := range users {
		ids[i] = OntologyIDFromUser(&u)
	}
	return ids
}

func KeyFromOntologyID(id ontology.ID) (uuid.UUID, error) {
	return uuid.Parse(id.Key)
}

var OntologyTypeID = ontology.ID{Type: OntologyType, Key: ""}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"username":   zyn.String(),
	"first_name": zyn.String(),
	"last_name":  zyn.String(),
	"root_user":  zyn.Bool(),
})

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	uuidKey, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var u User
	err = s.NewRetrieve().Entry(&u).WhereKeys(uuidKey).Exec(ctx, tx)
	if err != nil {
		return ontology.Resource{}, err
	}
	return newResource(u), err
}

type change = changex.Change[uuid.UUID, User]

func translateChange(ch change) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, User]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, User](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, User](s.cfg.DB).OpenNexter(ctx)
	return xiter.Map(n, newResource), closer, err
}

func newResource(u User) ontology.Resource {
	return core.NewResource(schema, OntologyID(u.Key), u.Username, u)
}
