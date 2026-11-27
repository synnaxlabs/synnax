// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"fmt"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// storedMethods returns the method of each endpoint of the config of type E that snap
// stores at key, by endpoint key. It returns no methods when snap stores no such
// config.
func storedMethods[E gorp.Entry[uuid.UUID]](
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
	methods func(E) map[string]StoredMethod,
) (map[string]StoredMethod, error) {
	var stored E
	err := gorp.NewRetrieve[uuid.UUID, E]().
		Where(gorp.MatchKeys[uuid.UUID, E](key)).
		Entry(&stored).
		Exec(ctx, snap)
	if errors.Is(err, query.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return methods(stored), nil
}

// checkMethods refuses an endpoint whose method a new endpoint cannot use, unless the
// stored config already gives the endpoint with the same key that method. An existing
// TRACE or CONNECT endpoint therefore still saves, and no endpoint can change to one.
func checkMethods(
	path string,
	stored map[string]StoredMethod,
	keys []string,
	methods []StoredMethod,
) error {
	v := validate.New(path)
	for i, m := range methods {
		v.Ternaryf(
			fmt.Sprintf("endpoints.%d.method", i),
			m.IsValid() && !Method(m).IsValid() && stored[keys[i]] != m,
			"%s is not supported for a new or changed endpoint",
			m,
		)
	}
	return v.Error()
}

func readMethods(c ReadConfig) map[string]StoredMethod {
	m := make(map[string]StoredMethod, len(c.Endpoints))
	for _, ep := range c.Endpoints {
		m[ep.Key] = ep.Method
	}
	return m
}

func writeMethods(c WriteConfig) map[string]StoredMethod {
	m := make(map[string]StoredMethod, len(c.Endpoints))
	for _, ep := range c.Endpoints {
		m[ep.Key] = ep.Method
	}
	return m
}

func checkReadMethods(ctx context.Context, snap gorp.Snapshot, c *ReadConfig) error {
	stored, err := storedMethods(ctx, snap, c.Key, readMethods)
	if err != nil {
		return err
	}
	keys := make([]string, len(c.Endpoints))
	methods := make([]StoredMethod, len(c.Endpoints))
	for i, ep := range c.Endpoints {
		keys[i], methods[i] = ep.Key, ep.Method
	}
	return checkMethods("http.read", stored, keys, methods)
}

func checkWriteMethods(ctx context.Context, snap gorp.Snapshot, c *WriteConfig) error {
	stored, err := storedMethods(ctx, snap, c.Key, writeMethods)
	if err != nil {
		return err
	}
	keys := make([]string, len(c.Endpoints))
	methods := make([]StoredMethod, len(c.Endpoints))
	for i, ep := range c.Endpoints {
		keys[i], methods[i] = ep.Key, ep.Method
	}
	return checkMethods("http.write", stored, keys, methods)
}
