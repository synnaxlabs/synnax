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
	"slices"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
	"go.uber.org/zap"
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

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeUser }

// SearchableFields implements ontology.SearchableFieldsProvider.
func (s *Service) SearchableFields() []string {
	return []string{"username", "first_name", "last_name"}
}

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	uuidKey, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	users := make([]User, 1)
	if err = s.NewRetrieve().
		Entry(&users[0]).
		Where(MatchKeys(uuidKey)).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	if err = s.ResolveUsernames(ctx, tx, users); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(users[0]), nil
}

type change = xchange.Change[Key, User]

// OnChange implements ontology.Service. A change to a user's credentials is reported as
// a change to the user, because the resource name can fall back to the username.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	disconnectUsers := s.table.Observe().OnChange(
		func(ctx context.Context, reader gorp.TxReader[Key, User]) {
			changes := slices.Collect(reader)
			if err := s.resolveChangedUsernames(ctx, changes); err != nil {
				s.cfg.L.Error(
					"failed to resolve usernames of changed users",
					zap.Error(err),
				)
			}
			f(ctx, xiter.Map(slices.Values(changes), translateChange))
		},
	)
	disconnectAuth := s.cfg.Auth.OnChange(
		func(ctx context.Context, keys iter.Seq[Key]) {
			users, err := s.retrieveWithUsernames(ctx, slices.Collect(keys))
			if err != nil {
				s.cfg.L.Error(
					"failed to retrieve users of changed credentials",
					zap.Error(err),
				)
				return
			}
			f(ctx, xiter.Map(slices.Values(users), func(u User) ontology.Change {
				return translateChange(
					change{Key: u.Key, Value: u, Variant: xchange.VariantSet},
				)
			}))
		},
	)
	return func() {
		disconnectUsers()
		disconnectAuth()
	}
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	usernames, err := s.cfg.Auth.Usernames(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, func(u User) ontology.Resource {
		u.Username = usernames[u.Key]
		return newResource(u)
	}), closer, nil
}

func (s *Service) resolveChangedUsernames(ctx context.Context, changes []change) error {
	var keys []Key
	for _, ch := range changes {
		if ch.Variant == xchange.VariantSet {
			keys = append(keys, ch.Key)
		}
	}
	usernames, err := s.cfg.Auth.UsernamesByKey(ctx, nil, keys...)
	if err != nil {
		return err
	}
	for i := range changes {
		changes[i].Value.Username = usernames[changes[i].Key]
	}
	return nil
}

// retrieveWithUsernames returns the users with the given keys that still exist.
func (s *Service) retrieveWithUsernames(
	ctx context.Context,
	keys []Key,
) ([]User, error) {
	var users []User
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&users).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	if err := s.ResolveUsernames(ctx, nil, users); err != nil {
		return nil, err
	}
	return users, nil
}

func translateChange(ch change) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key).String(),
		Value:   newResource(ch.Value),
	}
}

// newResource names the resource by the user's full name, falling back to the username.
func newResource(u User) ontology.Resource {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		name = u.Username
	}
	return ontology.NewResource(schema, u.OntologyID(), name, u)
}
