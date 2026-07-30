// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v1 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// lastStateVersion is the final Console state version ("1.0.0" files). A v1
// state parks the structural model under pendingUpload; v0 states embed it
// inline and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 1

var _ imex.ImportExporter = (*Service)(nil)

// Export retrieves the table identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no table exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var t Table
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: t.Name}
	if err = imex.Encode(&env, t); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a Table and persists it on tx, returning the
// ontology.ID of the newly-created table. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. When
// opts.Project is non-zero the table is created within that project exactly as a
// regular create would be. Envelopes older than Version are Console-era files —
// camelCase typed exports or console states — and are lifted forward; an envelope
// newer than Version is rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	t, err := s.decodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	t.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	t.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, opts.Project, &t); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(t.Key), nil
}

// stateV1 is the slice of the v1 Console state the importer needs: the structural
// model parked under pendingUpload when a table was never uploaded. The tag is
// camelCase because the file was written by the Console.
type stateV1 struct {
	PendingUpload *stateV1Document `json:"pendingUpload" msgpack:"pendingUpload"`
}

// stateV1Document mirrors the typed Table body fields as they appear inside a v1
// Console state's pendingUpload.
type stateV1Document struct {
	Rows    []Row           `json:"rows"`
	Columns []Column        `json:"columns"`
	Cells   map[string]Cell `json:"cells"`
}

func (s *Service) decodeImport(ctx context.Context, env imex.Envelope) (Table, error) {
	switch {
	case env.Version > Version:
		return Table{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	case env.Version == Version:
		return imex.Decode[Table](ctx, env)
	}
	named, err := imex.BodyNamed(ctx, env)
	if err != nil {
		return Table{}, err
	}
	// Console-era typed exports ("1.0.0"-stamped or versionless) carry the current
	// shape with camelCase keys; every Table field key is a single word, so the
	// standard decoder's case-insensitive matching covers them. Console states
	// never carry a name.
	if named {
		return imex.Decode[Table](ctx, env)
	}
	if env.Version == lastStateVersion {
		st, err := imex.Decode[stateV1](ctx, env)
		if err != nil {
			return Table{}, err
		}
		if st.PendingUpload == nil {
			return Table{}, errors.Wrap(
				validate.ErrValidation, "table file has no structural data",
			)
		}
		p := st.PendingUpload
		return Table{Rows: p.Rows, Columns: p.Columns, Cells: p.Cells}, nil
	}
	// v0 console states embed the structural model inline: ride the storage lift,
	// which decodes the body through the legacy chain.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Table{}, err
	}
	return v2.MigrateTable(ctx, v1.Table{Name: env.Name, Data: body})
}
