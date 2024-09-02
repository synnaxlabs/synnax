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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("user")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

type Service struct {
	Config
	group group.Group
}

const groupName = "Users"

func NewService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:  gorp.OverrideTx(s.DB, tx),
		otg: s.Ontology.NewWriter(tx),
		svc: s,
	}
}

// Retrieve retrieves a User by its key.
func (s *Service) Retrieve(ctx context.Context, key uuid.UUID) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		WhereKeys(key).
		Entry(&u).
		Exec(ctx, s.DB)
}

// RetrieveByUsername retrieves a User by its username.
func (s *Service) RetrieveByUsername(ctx context.Context, username string) (User, error) {
	var u User
	return u, gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exec(ctx, s.DB)
}

// UsernameExists checks if a user with the given username exists.
func (s *Service) UsernameExists(ctx context.Context, username string) (bool, error) {
	var u User
	return gorp.NewRetrieve[uuid.UUID, User]().
		Where(func(u *User) bool { return u.Username == username }).
		Entry(&u).
		Exists(ctx, s.DB)
}

type Writer struct {
	svc *Service
	tx  gorp.Tx
	otg ontology.Writer
}

func (w Writer) Create(ctx context.Context, u *User) error {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}

	exists, err := w.svc.UsernameExists(ctx, u.Username)
	if err != nil {
		return err
	}
	if exists {
		return query.UniqueViolation
	}

	if err := gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(ctx, w.tx); err != nil {
		return err
	}

	otgID := OntologyID(u.Key)

	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}

	return w.otg.DefineRelationship(ctx, w.svc.group.OntologyID(), ontology.ParentOf, otgID)
}

func (w Writer) Update(ctx context.Context, u User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entry(&u).Exec(ctx, w.tx)
}
