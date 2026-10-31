// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// APIKey is an API key as the API returns it: the stored record without its hash.
type APIKey struct {
	// Key is the unique identifier for this API key.
	Key apikey.Key `json:"key" msgpack:"key"`
	// Subject is the user or rack the key logs in as.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Name is a label that tells the subject's keys apart.
	Name string `json:"name" msgpack:"name"`
	// Created is when the key was created.
	Created telem.TimeStamp `json:"created" msgpack:"created"`
}

func apiKey(k apikey.APIKey) APIKey {
	return APIKey{Key: k.Key, Subject: k.Subject, Name: k.Name, Created: k.Created}
}

// KeyCreateRequest creates an API key.
type KeyCreateRequest struct {
	// Subject is the user or rack the key logs in as. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Name is a label that tells the subject's keys apart.
	Name string `json:"name" msgpack:"name"`
}

// KeyCreateResponse is a new API key.
type KeyCreateResponse struct {
	// Token is the API key to log in with. It is the only copy of its secret.
	Token string `json:"token" msgpack:"token"`
	// Key is the stored key.
	Key APIKey `json:"key" msgpack:"key"`
}

// CreateKey creates an API key for the caller, or for another user or rack when the
// caller may create that subject. It returns a validation error when the subject is not
// a user or rack, and an error wrapping query.ErrNotFound when it does not exist.
func (s *Service) CreateKey(
	ctx context.Context,
	tx gorp.Tx,
	req KeyCreateRequest,
) (KeyCreateResponse, error) {
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser &&
		subject.Type != ontology.ResourceTypeRack {
		return KeyCreateResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"an API key cannot log in as a %s",
			subject.Type,
		)
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionCreate,
		[]ontology.ID{subject},
	); err != nil {
		return KeyCreateResponse{}, err
	}
	exists, err := s.ontology.NewRetrieve().WhereIDs(subject).Exists(ctx, tx)
	if err != nil {
		return KeyCreateResponse{}, err
	}
	if !exists {
		return KeyCreateResponse{}, errors.Wrapf(query.ErrNotFound, "%s", subject)
	}
	k, token, err := s.apiKey.NewWriter(tx).Create(ctx, subject, req.Name)
	if err != nil {
		return KeyCreateResponse{}, err
	}
	return KeyCreateResponse{Token: token, Key: apiKey(k)}, nil
}

// KeyRetrieveRequest selects API keys to list.
type KeyRetrieveRequest struct {
	// Subjects limits the result to these subjects' keys. Empty means the caller.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// KeyRetrieveResponse lists API keys without their hashes.
type KeyRetrieveResponse struct {
	// Keys are the keys that matched.
	Keys []APIKey `json:"keys" msgpack:"keys"`
}

// RetrieveKeys lists the caller's API keys, or the keys of the given subjects when the
// caller may retrieve those subjects.
func (s *Service) RetrieveKeys(
	ctx context.Context,
	req KeyRetrieveRequest,
) (KeyRetrieveResponse, error) {
	caller := GetSubject(ctx)
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []ontology.ID{caller}
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionRetrieve,
		subjects,
	); err != nil {
		return KeyRetrieveResponse{}, err
	}
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchSubjects(subjects...)).
		Entries(&keys).
		Exec(ctx, nil); err != nil {
		return KeyRetrieveResponse{}, err
	}
	return KeyRetrieveResponse{Keys: lo.Map(keys, func(k apikey.APIKey, _ int) APIKey {
		return apiKey(k)
	})}, nil
}

// KeyDeleteRequest names API keys to delete.
type KeyDeleteRequest struct {
	// Keys are the API keys to delete.
	Keys []apikey.Key `json:"keys" msgpack:"keys"`
}

// DeleteKeys deletes the given API keys and ends the sessions they opened. A caller may
// delete its own keys; deleting another subject's needs permission to delete that
// subject. Unknown keys are ignored.
func (s *Service) DeleteKeys(
	ctx context.Context,
	tx gorp.Tx,
	req KeyDeleteRequest,
) (struct{}, error) {
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchKeys(req.Keys...)).
		Entries(&keys).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	subjects := lo.Uniq(lo.Map(keys, func(k apikey.APIKey, _ int) ontology.ID {
		return k.Subject
	}))
	if err := s.enforceOthers(
		ctx,
		GetSubject(ctx),
		access.ActionDelete,
		subjects,
	); err != nil {
		return struct{}{}, err
	}
	stored := lo.Map(keys, func(k apikey.APIKey, _ int) apikey.Key { return k.Key })
	if err := s.apiKey.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, stored...)
}
