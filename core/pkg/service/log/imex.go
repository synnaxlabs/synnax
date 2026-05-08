// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
) (string, error) {
	// In the future, when the log is strongly-typed in ORC, this call will return a
	// Log directly and we can stamp the name + a fresh key onto it before handing it
	// to the writer.
	migrated, err := migrations.Migrate(env.Version, env.Data)
	if err != nil {
		return "", err
	}
	l := Log{
		Key:  uuid.New(),
		Name: env.Name,
		Data: msgpack.EncodedJSON(migrated.ToMap()),
	}
	if err := s.NewWriter(tx).Create(ctx, uuid.Nil, &l); err != nil {
		return "", err
	}
	return l.Key.String(), nil
}

func (s *Service) Export(ctx context.Context, key string) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&l).Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	// In the future, when the log is strongly-typed in ORC, we can just return an
	// imex.Envelope at this point with the Data field determined by an Oracle-generated
	// method log.Data() that returns a map[string]any.
	var d migrations.Latest
	if l.Data != nil {
		if err := l.Data.Unmarshal(&d); err != nil {
			return imex.Envelope{}, err
		}
	}
	return imex.Envelope{
		Version: migrations.LatestVersion,
		Type:    string(ontology.ResourceTypeLog),
		Name:    l.Name,
		Data:    d.ToMap(),
	}, nil
}
