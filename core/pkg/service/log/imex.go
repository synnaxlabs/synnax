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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	env imex.Envelope,
) error {
	migrated, err := s.migrateData(env.Version, env.Data)
	if err != nil {
		return err
	}
	key, err := uuid.Parse(env.Key)
	if err != nil {
		key = uuid.New()
	}
	wsKey, err := uuid.Parse(parent.Key)
	if err != nil {
		return err
	}
	name := env.Name
	if name == "" {
		name = "Imported Log"
	}
	dumped, err := v1.Schema.Dump(migrated)
	if err != nil {
		return err
	}
	dumpedMap, ok := dumped.(map[string]any)
	if !ok {
		return errors.New("unexpected dump result type")
	}
	l := Log{Key: key, Name: name, Data: dumpedMap}
	return s.NewWriter(tx).Create(ctx, wsKey, &l)
}

func (s *Service) migrateData(version int, data map[string]any) (v1.Data, error) {
	switch {
	case version >= v1.Version:
		var d v1.Data
		if err := v1.Schema.Parse(data, &d); err != nil {
			return v1.Data{}, err
		}
		return d, nil
	case version >= v0.Version:
		var d v0.Data
		if err := v0.Schema.Parse(data, &d); err != nil {
			return v1.Data{}, err
		}
		return v1.Migrate(d)
	default:
		return v1.Data{}, errors.Newf("unknown log data version %d", version)
	}
}

func (s *Service) Export(
	ctx context.Context,
	tx gorp.Tx,
	key string,
) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err := s.NewRetrieve().WhereKeys(k).Entry(&l).Exec(ctx, tx); err != nil {
		return imex.Envelope{}, err
	}
	data := l.Data
	if data == nil {
		data = make(map[string]any)
	}
	data["key"] = l.Key.String()
	data["name"] = l.Name
	return imex.Envelope{
		Type: string(ontology.ResourceTypeLog),
		Key:  l.Key.String(),
		Name: l.Name,
		Data: data,
	}, nil
}
