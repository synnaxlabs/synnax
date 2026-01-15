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
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Service is the core authentication service for the Synnax API.
type Service struct {
	db            *gorp.DB
	authenticator svcauth.Authenticator
	token         *token.Service
	access        *rbac.Service
	internal      *user.Service
}

// NewService creates a new Service that allows for registering, updating, and
// removing users.
func NewService(cfg config.Config) *Service {
	return &Service{
		db:            cfg.Distribution.DB,
		authenticator: cfg.Service.Auth,
		token:         cfg.Service.Token,
		access:        cfg.Service.RBAC,
		internal:      cfg.Service.User,
	}
}

// NewUser holds information for creating a new user in a Synnax server. The username
// and password are required, and the first and last name are optional.
type NewUser struct {
	svcauth.InsecureCredentials
	FirstName string    `json:"first_name" msgpack:"first_name"`
	LastName  string    `json:"last_name" msgpack:"last_name"`
	Key       uuid.UUID `json:"key" msgpack:"key"`
}

type (
	CreateRequest struct {
		Users []NewUser `json:"users" msgpack:"users"`
	}
	CreateResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Create registers the new users with the provided credentials. If successful, Create
// returns a slice of the new users.
func (svc *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: user.OntologyType}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var res CreateResponse
	return res, svc.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.NewWriter(tx)
		newUsers := make([]user.User, len(req.Users))
		for i, u := range req.Users {
			if err := svc.authenticator.NewWriter(tx).Register(ctx, u.InsecureCredentials); err != nil {
				return err
			}
			newUsers[i].Username = u.Username
			newUsers[i].FirstName = u.FirstName
			newUsers[i].LastName = u.LastName
			newUsers[i].Key = u.Key
			if err := w.Create(ctx, &newUsers[i]); err != nil {
				return err
			}

		}
		res.Users = newUsers
		return nil
	})
}

type ChangeUsernameRequest struct {
	Key      uuid.UUID `json:"key" msgpack:"key"`
	Username string    `json:"username" msgpack:"username"`
}

// ChangeUsername changes the username for the user with the given key.
func (s *Service) ChangeUsername(ctx context.Context, req ChangeUsernameRequest) (types.Nil, error) {
	subject := auth.GetSubject(ctx)
	if subject.Key == req.Key.String() {
		return types.Nil{}, errors.New("you cannot change your own username through the user service")
	}
	var u user.User
	if err := s.internal.NewRetrieve().WhereKeys(req.Key).Entry(&u).Exec(ctx, nil); err != nil {
		return types.Nil{}, err
	}
	if u.Username == req.Username {
		return types.Nil{}, nil
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.authenticator.NewWriter(tx).InsecureUpdateUsername(
			ctx,
			u.Username,
			req.Username,
		); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).ChangeUsername(ctx, req.Key, req.Username)
	})
}

type RenameRequest struct {
	Key       uuid.UUID `json:"key" msgpack:"key"`
	FirstName string    `json:"first_name" msgpack:"first_name"`
	LastName  string    `json:"last_name" msgpack:"last_name"`
}

// Rename changes the name for the user with the provided key. If either the first
// or last name is empty, the corresponding field will not be updated.
func (s *Service) Rename(ctx context.Context, req RenameRequest) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).ChangeName(ctx, req.Key, req.FirstName, req.LastName)
	})
}

type (
	RetrieveRequest struct {
		Keys      []uuid.UUID `json:"keys" msgpack:"keys"`
		Usernames []string    `json:"usernames" msgpack:"usernames"`
	}
	RetrieveResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Retrieve returns the users with the provided keys or usernames.
func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()

	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if len(req.Usernames) > 0 {
		q = q.WhereUsernames(req.Usernames...)
	}
	var users []user.User
	if err := q.Entries(&users).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: user.OntologyIDsFromUsers(users),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Users: users}, nil
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

// Delete removes the users with the provided keys from the Synnax cluster.
func (s *Service) Delete(ctx context.Context, req DeleteRequest) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: user.OntologyIDsFromKeys(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}

	// we have to retrieve individually here in case one of the users is already deleted
	users := make([]user.User, 0, len(req.Keys))
	for _, key := range req.Keys {
		var u user.User
		err := s.internal.NewRetrieve().WhereKeys(key).Entry(&u).Exec(ctx, nil)
		if err != nil && !errors.Is(err, query.NotFound) {
			return types.Nil{}, err
		}
		users = append(users, u)
	}

	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.authenticator.NewWriter(tx).
			InsecureDeactivate(ctx, lo.Map(users, func(u user.User, _ int) string {
				return u.Username
			})...); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
