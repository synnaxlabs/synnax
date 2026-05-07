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
	payload imex.ImportPayload,
) (string, error) {
	migrated, err := migrateData(payload.Version, payload.Data)
	if err != nil {
		return "", err
	}
	name := payload.Name
	if name == "" {
		name = "Imported Log"
	}
	data, err := encodedJSONFromStruct(migrated)
	if err != nil {
		return "", err
	}
	l := Log{Key: uuid.New(), Name: name, Data: data}
	if err := s.NewWriter(tx).Create(ctx, uuid.Nil, &l); err != nil {
		return "", err
	}
	return l.Key.String(), nil
}

func migrateData(version int, data map[string]any) (v1.Data, error) {
	switch {
	case version > v1.Version:
		return v1.Data{}, errors.Newf(
			"log version %d is newer than this Core supports (latest: %d)",
			version, v1.Version,
		)
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
		return v1.Data{}, errors.Newf("unknown log version %d", version)
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
	d.Key = l.Key.String()
	d.Name = l.Name
	raw, err := json.Marshal(d)
	if err != nil {
		return imex.Envelope{}, err
	}
	return imex.Envelope{
		Version: v1.Version,
		Type:    string(ontology.ResourceTypeLog),
		Key:     l.Key.String(),
		Name:    l.Name,
		Data:    raw,
	}, nil
}

// encodedJSONFromStruct bridges from a typed migration struct into the
// msgpack.EncodedJSON form used by the storage layer. It round-trips through
// JSON because EncodedJSON is a map[string]any; byte-level fidelity end to end
// is future work that would replace EncodedJSON with json.RawMessage.
func encodedJSONFromStruct(v any) (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m msgpack.EncodedJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
