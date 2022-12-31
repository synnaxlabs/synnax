// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/google/uuid"
)

type Service struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

func (s *Service) NewWriter() Writer { return s.NewWriterUsingTxn(s.DB) }

func (s *Service) NewWriterUsingTxn(txn gorp.Txn) Writer {
	return Writer{
		txn:     txn,
		Service: s,
	}
}

func (s *Service) Retrieve(key uuid.UUID) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		WhereKeys(key).
		Entry(&u).
		Exec(s.DB)
}

func (s *Service) RetrieveByUsername(username string) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exec(s.DB)
}

func (s *Service) UsernameExists(username string) (bool, error) {
	var u User
	return gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exists(s.DB)
}

type Writer struct {
	*Service
	txn gorp.Txn
}

func (w Writer) Create(u *User) error {

	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}

	exists, err := w.UsernameExists(u.Username)
	if err != nil {
		return err
	}
	if exists {
		return query.UniqueViolation
	}

	if err = w.Ontology.NewWriterUsingTxn(w.txn).
		DefineResource(OntologyID(u.Key)); err != nil {
		return err
	}

	return gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(w.txn)
}

func (w Writer) Update(u User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entry(&u).Exec(w.txn)
}
