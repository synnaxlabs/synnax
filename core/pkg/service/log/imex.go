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
	"encoding/json"

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
	migrated, err := migrateData(env.Version, env.Data)
	if err != nil {
		return "", err
	}
	name := env.Name
	if name == "" {
		name = "Imported Log"
	}
	data, err := structToEncodedJSON(migrated)
	if err != nil {
		return "", err
	}
	l := Log{Key: uuid.New(), Name: name, Data: data}
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

func (s *Service) Export(
	ctx context.Context,
	key string,
) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&l).Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	var d v1.Data
	if l.Data != nil {
		if err := l.Data.Unmarshal(&d); err != nil {
			return imex.Envelope{}, errors.Wrap(err, "decode stored log data")
		}
	}
	data, err := structToMap(d)
	if err != nil {
		return imex.Envelope{}, err
	}
	return imex.Envelope{
		Version: v1.Version,
		Type:    string(ontology.ResourceTypeLog),
		Name:    l.Name,
		Data:    data,
	}, nil
}

// structToMap converts a typed struct into a generic map[string]any by round-tripping
// through JSON, matching the JSON struct tags that govern the public wire shape.
func structToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// structToEncodedJSON bridges a typed migration struct into the msgpack.EncodedJSON form
// used by the storage layer. EncodedJSON is itself a map[string]any; byte-level fidelity
// end to end is future work that would replace EncodedJSON with json.RawMessage.
func structToEncodedJSON(v any) (msgpack.EncodedJSON, error) {
	m, err := structToMap(v)
	if err != nil {
		return nil, err
	}
	return msgpack.EncodedJSON(m), nil
}
