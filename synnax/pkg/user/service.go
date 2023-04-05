// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

func (s *Service) NewWriter(txn gorp.TypedWriter) Writer {
	return Writer{
		txn:     txn,
		Service: s,
	}
}

func (s *Service) Retrieve(ctx context.Context, key uuid.UUID) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		WhereKeys(key).
		Entry(&u).
		Exec(s.DB.NewReader(ctx))
}

func (s *Service) RetrieveByUsername(ctx context.Context, username string) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exec(s.DB.NewReader(ctx))
}

func (s *Service) UsernameExists(ctx context.Context, username string) (bool, error) {
	var u User
	return gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exists(s.DB.NewReader(ctx))
}

type Writer struct {
	*Service
	writer gorp.Writer
}

func (w Writer) Create(ctx context.Context, u *User) error {

	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}

	exists, err := w.UsernameExists(ctx, u.Username)
	if err != nil {
		return err
	}
	if exists {
		return query.UniqueViolation
	}

	if err = w.Ontology.NewWriter(w.writer).
		DefineResource(ctx, OntologyID(u.Key)); err != nil {
		return err
	}

	return gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(w.writer)
}

func (w Writer) Update(ctx context.Context, u User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entry(&u).Exec(w.writer)
}
