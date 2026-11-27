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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates and deletes API keys in one transaction. It makes no identity check
// and ends no sessions: the caller authorizes the change and revokes the sessions of
// deleted keys.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create stores a new API key for subject and returns it with the full key, which is
// the only copy of its secret. A zero expires becomes [telem.TimeStampMax], which
// never expires. It returns a validation error when subject is invalid or expires is
// not in the future.
func (w Writer) Create(
	ctx context.Context,
	subject resource.ID,
	name string,
	expires telem.TimeStamp,
) (APIKey, string, error) {
	if expires == 0 {
		expires = telem.TimeStampMax
	}
	k := APIKey{
		Key:     uuid.New(),
		Subject: subject,
		Name:    name,
		Created: w.svc.now(),
		Expires: expires,
	}
	if err := k.Validate(); err != nil {
		return APIKey{}, "", err
	}
	v := validate.New("api_key")
	v.Ternary(
		"expires",
		expires <= k.Created,
		"must be in the future",
	)
	if err := v.Error(); err != nil {
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

// Rotate gives the API key a new secret, creation time, and expiry, and returns the key
// with its new token. The old token stops working, but the key keeps its key, subject,
// and name, so a delete of the key still ends it. The key gets [Service.LongestExpiry]
// when the service has a maximum lifetime, and otherwise lasts as long as before, so a
// key that never expires still never expires. Rotate returns
// [auth.ErrInvalidCredentials] when the key has expired, and an error wrapping
// [query.ErrNotFound] when it does not exist.
func (w Writer) Rotate(ctx context.Context, key Key) (APIKey, string, error) {
	var k APIKey
	if err := w.svc.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&k).
		Exec(ctx, w.tx); err != nil {
		return APIKey{}, "", err
	}
	if w.svc.expired(k) {
		return APIKey{}, "", errors.Wrap(auth.ErrInvalidCredentials, "API key expired")
	}
	token, hash, err := auth.NewToken(prefix, k.Key)
	if err != nil {
		return APIKey{}, "", err
	}
	k.Expires = w.svc.renewedExpiry(k)
	k.Hash, k.Created = hash, w.svc.now()
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
