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
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
)

// APIKey is an API key as the API returns it: the stored record without its hash.
type APIKey struct {
	// Key is the unique identifier for this API key.
	Key apikey.Key `json:"key" msgpack:"key"`
	// Subject is the user or rack the key logs in as.
	Subject resource.ID `json:"subject" msgpack:"subject"`
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
	Subject resource.ID `json:"subject" msgpack:"subject"`
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

// CreateKey creates an API key for the caller, or for another user or rack. It returns
// a validation error when the subject is not a user or rack that exists.
func (s *Service) CreateKey(
	ctx context.Context,
	tx gorp.Tx,
	req KeyCreateRequest,
) (KeyCreateResponse, error) {
	subject := lo.Ternary(req.Subject.IsZero(), GetSubject(ctx), req.Subject)
	k, token, err := s.apiKey.NewWriter(tx).Create(ctx, subject, req.Name)
	if err != nil {
		return KeyCreateResponse{}, err
	}
	return KeyCreateResponse{Token: token, Key: apiKey(k)}, nil
}

// KeyRetrieveRequest selects API keys to list.
type KeyRetrieveRequest struct {
	// Subjects limits the result to these subjects' keys. Empty means the caller.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// KeyRetrieveResponse lists API keys without their hashes.
type KeyRetrieveResponse struct {
	// Keys are the keys that matched.
	Keys []APIKey `json:"keys" msgpack:"keys"`
}

// Holders implements bind.Held.
func (r KeyRetrieveRequest) Holders() []resource.ID { return r.Subjects }

// RetrieveKeys lists the API keys of the given subjects in snap, or the caller's keys
// when the request names no subject.
func (s *Service) RetrieveKeys(
	ctx context.Context,
	snap gorp.Snapshot,
	req KeyRetrieveRequest,
) (KeyRetrieveResponse, error) {
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []resource.ID{GetSubject(ctx)}
	}
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchSubjects(subjects...)).
		Entries(&keys).
		Exec(ctx, snap); err != nil {
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

// DeleteKeys deletes the given API keys and ends the sessions they opened. Unknown keys
// are ignored.
func (s *Service) DeleteKeys(
	ctx context.Context,
	tx gorp.Tx,
	req KeyDeleteRequest,
) (struct{}, error) {
	if err := s.apiKey.NewWriter(tx).Delete(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, req.Keys...)
}
