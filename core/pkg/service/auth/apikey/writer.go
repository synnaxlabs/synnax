// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates and deletes API keys in one transaction. It makes no identity check
// and ends no sessions: the caller authorizes the change and revokes the sessions of
// deleted keys.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create stores a new API key for subject and returns it with the full key, which is
// the only copy of its secret. It returns a validation error when subject is invalid.
func (w Writer) Create(
	ctx context.Context,
	subject ontology.ID,
	name string,
) (APIKey, string, error) {
	k := APIKey{Key: uuid.New(), Subject: subject, Name: name, Created: telem.Now()}
	if err := k.Validate(); err != nil {
		return APIKey{}, "", err
	}
	token, hash, err := auth.NewToken(prefix, k.Key)
	if err != nil {
		return APIKey{}, "", err
	}
	k.Hash = hash
	if err = w.svc.table.NewCreate().Entry(&k).Exec(ctx, w.tx); err != nil {
		return APIKey{}, "", err
	}
	return k, token, nil
}

// Delete removes the API keys with the given keys. Unknown keys are ignored.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, APIKey](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) deleteFor(ctx context.Context, subjects ...ontology.ID) error {
	return w.svc.table.NewDelete().
		Where(MatchSubjects(subjects...)(w.svc.NewRetrieve())).
		Exec(ctx, w.tx)
}
