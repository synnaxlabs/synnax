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
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
) (string, error) {
	// In the future, the migrateData function will just return a Log. We can then set
	// the name field on the log and a uuid.New() key and create the log in the
	// database.
	migrated, err := migrateData(env.Version, env.Data)
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

func migrateData(version imex.Version, data map[string]any) (v1.Data, error) {
	switch {
	case version > v1.Version:
		return v1.Data{}, errors.Newf(
			"log version %d is newer than this Core supports (latest: %d)",
			version, v1.Version,
		)
	case version == v1.Version:
		var d v1.Data
		if err := v1.Schema.Parse(data, &d); err != nil {
			return v1.Data{}, err
		}
		return d, nil
	default:
		var d v0.Data
		if err := v0.Schema.Parse(data, &d); err != nil {
			return v1.Data{}, err
		}
		return v1.Migrate(d)
	}
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
	var d v1.Data
	if l.Data != nil {
		if err := l.Data.Unmarshal(&d); err != nil {
			return imex.Envelope{}, err
		}
	}
	return imex.Envelope{
		Version: v1.Version,
		Type:    string(ontology.ResourceTypeLog),
		Name:    l.Name,
		Data:    d.ToMap(),
	}, nil
}
