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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v55 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v55"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// Version is the per-schema version stamped on every exported log envelope and the
// highest version the importer accepts.
const Version imex.Version = 2

var _ imex.ImportExporter = (*Service)(nil)

// Export retrieves the log identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no log exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&l).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: l.Name}
	if err = imex.Encode(&env, l); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a Log and persists it on tx, returning the
// ontology.ID of the newly-created log. The exported key is discarded and a fresh one
// is generated so that importing always materializes a new resource rather than
// overwriting an existing log with a colliding key. Envelopes older than Version are
// legacy camelCase Console exports and are lifted forward through the migration chain;
// an envelope newer than Version is rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
) (ontology.ID, error) {
	l, err := s.decodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	l.Key = uuid.Nil
	if err = s.NewWriter(tx).Create(ctx, uuid.Nil, &l); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(l.Key), nil
}

func (s *Service) decodeImport(ctx context.Context, env imex.Envelope) (Log, error) {
	switch {
	case env.Version == Version:
		return imex.Decode[Log](ctx, env)
	case env.Version < Version:
		body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
		if err != nil {
			return Log{}, err
		}
		return MigrateLog(ctx, v55.Log{Name: env.Name, Data: body})
	default:
		return Log{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	}
}
