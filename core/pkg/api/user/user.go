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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Service is the core authentication service for the Synnax API.
type Service struct {
	access   *rbac.Service
	internal *user.Service
	auth     *svcauth.Service
}

// NewService creates a new Service that allows for registering, updating, and removing
// users.
func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Service.User,
		auth:     cfg.Service.Auth,
	}, nil
}

// NewUser is the create-request payload for a single user. It bundles the user-record
// fields and the credentials the auth service will store.
type NewUser struct {
	svcauth.Credentials
	FirstName string   `json:"first_name" msgpack:"first_name"`
	LastName  string   `json:"last_name"  msgpack:"last_name"`
	Key       user.Key `json:"key"        msgpack:"key"`
}

type (
	CreateRequest struct {
		Users []NewUser `json:"users" msgpack:"users"`
	}
	CreateResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Create registers the new users with the provided credentials. A user whose Key
// already belongs to an existing user is updated in place instead: its username,
// password, and names are replaced with the request values, and the subject must hold
// update access on that user. Empty names leave the existing values unchanged, as in
// [Service.Rename]. If successful, Create returns the created or updated users.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeUser}},
	}); err != nil {
		return CreateResponse{}, err
	}
	authW := s.auth.NewWriter(tx)
	userW := s.internal.NewWriter(tx)
	newUsers := make([]user.User, len(req.Users))
	for i, nu := range req.Users {
		existing, found, err := s.retrieveByKey(ctx, tx, nu.Key)
		if err != nil {
			return CreateResponse{}, err
		}
		if found {
			if newUsers[i], err = s.update(ctx, tx, existing, nu); err != nil {
				return CreateResponse{}, err
			}
			continue
		}
		if err := authW.Register(ctx, nu.Credentials); err != nil {
			return CreateResponse{}, err
		}
		if newUsers[i], err = userW.Create(ctx, user.User{
			Username:  nu.Username,
			FirstName: nu.FirstName,
			LastName:  nu.LastName,
			Key:       nu.Key,
		}); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse{Users: newUsers}, nil
}

// retrieveByKey returns the user with the given key and whether one exists. A nil key
// never matches.
func (s *Service) retrieveByKey(
	ctx context.Context,
	tx gorp.Tx,
	key user.Key,
) (user.User, bool, error) {
	if key == uuid.Nil() {
		return user.User{}, false, nil
	}
	var u user.User
	err := s.internal.NewRetrieve().Where(user.MatchKeys(key)).Entry(&u).Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return user.User{}, false, nil
	}
	if err != nil {
		return user.User{}, false, err
	}
	return u, true, nil
}

// update replaces the username, password, and names of existing with the values in nu,
// keeping the user record and credentials in sync.
func (s *Service) update(
	ctx context.Context,
	tx gorp.Tx,
	existing user.User,
	nu NewUser,
) (user.User, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{existing.OntologyID()},
	}); err != nil {
		return user.User{}, err
	}
	authW, userW := s.auth.NewWriter(tx), s.internal.NewWriter(tx)
	if existing.Username != nu.Username {
		if err := userW.ChangeUsername(ctx, existing.Key, nu.Username); err != nil {
			return user.User{}, err
		}
		if err := authW.UpdateUsername(
			ctx, existing.Username, nu.Username,
		); err != nil {
			return user.User{}, err
		}
	}
	if err := authW.ChangePassword(ctx, nu.Credentials); err != nil {
		return user.User{}, err
	}
	if err := userW.ChangeName(
		ctx, existing.Key, nu.FirstName, nu.LastName,
	); err != nil {
		return user.User{}, err
	}
	var updated user.User
	return updated, s.internal.NewRetrieve().
		Where(user.MatchKeys(existing.Key)).Entry(&updated).Exec(ctx, tx)
}

type ChangeUsernameRequest struct {
	Username string   `json:"username" msgpack:"username"`
	Key      user.Key `json:"key"      msgpack:"key"`
}

// ChangeUsername changes the username for the user with the given key.
func (s *Service) ChangeUsername(
	ctx context.Context,
	tx gorp.Tx,
	req ChangeUsernameRequest,
) (struct{}, error) {
	subject := auth.GetSubject(ctx)
	if subject.Key == req.Key.String() {
		return struct{}{}, errors.New(
			"you cannot change your own username through the user service",
		)
	}
	var u user.User
	if err := s.internal.NewRetrieve().
		Where(user.MatchKeys(req.Key)).Entry(&u).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if u.Username == req.Username {
		return struct{}{}, nil
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.internal.NewWriter(tx).
		ChangeUsername(ctx, req.Key, req.Username); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).
		UpdateUsername(ctx, u.Username, req.Username)
}

type RenameRequest struct {
	FirstName string   `json:"first_name" msgpack:"first_name"`
	LastName  string   `json:"last_name"  msgpack:"last_name"`
	Key       user.Key `json:"key"        msgpack:"key"`
}

// Rename changes the name for the user with the provided key. If either the first or
// last name is empty, the corresponding field will not be updated.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		ChangeName(ctx, req.Key, req.FirstName, req.LastName)
}

type (
	RetrieveRequest struct {
		Keys      []user.Key `json:"keys"      msgpack:"keys"`
		Usernames []string   `json:"usernames" msgpack:"usernames"`
	}
	RetrieveResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Retrieve returns the users with the provided keys or usernames.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(user.MatchKeys(req.Keys...))
	}
	if len(req.Usernames) > 0 {
		q = q.Where(user.MatchUsernames(req.Usernames...))
	}
	var users []user.User
	if err := q.Entries(&users).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: user.OntologyIDsFromUsers(users),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Users: users}, nil
}

type DeleteRequest struct {
	Keys []user.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the users with the provided keys from the Synnax cluster.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: user.OntologyIDsFromKeys(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	// Look up the usernames of the keys that actually exist so we can deactivate the
	// matching auth rows. A bare-key retrieve wraps query.ErrNotFound when any key is
	// missing; we treat that as "those keys are simply not here" and continue with
	// whatever was found, so deleting a non-existent user is a no-op rather than an
	// error.
	var toDelete []user.User
	if err := s.internal.NewRetrieve().
		Where(user.MatchKeys(req.Keys...)).
		Entries(&toDelete).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	if err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	if len(toDelete) == 0 {
		return struct{}{}, nil
	}
	usernames := lo.Map(toDelete, func(u user.User, _ int) string {
		return u.Username
	})
	return struct{}{}, s.auth.NewWriter(tx).Deactivate(ctx, usernames...)
}
