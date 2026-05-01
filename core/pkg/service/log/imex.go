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
	data, err := encodedJSONFromStruct(migrated)
	if err != nil {
		return err
	}
	l := Log{Key: key, Name: name, Data: data}
	return s.NewWriter(tx).Create(ctx, wsKey, &l)
}

func (s *Service) migrateData(version int, raw json.RawMessage) (v1.Data, error) {
	switch {
	case version >= v1.Version:
		var d v1.Data
		if err := imex.Decode(raw, &d); err != nil {
			return v1.Data{}, err
		}
		if err := d.Validate(); err != nil {
			return v1.Data{}, err
		}
		return d, nil
	case version >= v0.Version:
		var d v0.Data
		if err := imex.Decode(raw, &d); err != nil {
			return v1.Data{}, err
		}
		if err := d.Validate(); err != nil {
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
		Type: string(ontology.ResourceTypeLog),
		Key:  l.Key.String(),
		Name: l.Name,
		Data: raw,
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
