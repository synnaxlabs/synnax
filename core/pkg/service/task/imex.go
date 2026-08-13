// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.Exporter = (*Service)(nil)

// Export retrieves the task identified by id and serializes it as an imex.Envelope. The
// task's type-specific config is merged flat into the envelope body — the driver reads
// a task file directly as its config, so the config is not nested under a `config`
// field; version, the fine-grained task type (e.g. "opc_read"), and name are stamped on
// top. The version is the config type's own number line, taken from its config store.
// Routing to this exporter still happens under the coarse "task" ontology type via
// Type. It returns query.ErrNotFound if no task exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var t Task
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	store, ok := s.cfg.Configs.Store(t.Type)
	if !ok {
		return imex.Envelope{}, errors.Newf(
			"no config store registered for task type %q", t.Type,
		)
	}
	body := map[string]any(t.Config)
	if body == nil {
		body = make(map[string]any)
	}
	body["type"] = t.Type
	body["name"] = t.Name
	env := imex.Envelope{Version: store.Version()}
	if err = imex.Encode(&env, body); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

var _ imex.Importer = (*Service)(nil)

// Match always reports false: task files always carry a type header, so a typeless
// body is never a task.
func (*Service) Match(map[string]any) bool { return false }

// Import decodes env as a task file — the flat body is the task's config with the
// type, name, and version headers stamped on top — and creates the task as a draft on
// no rack. The version dispatches through the type's config store: legacy versions
// upgrade through the same rewrites as the boot migration, and a version newer than
// the store's returns a validation error. A file with no version header is a legacy
// Console export at version zero.
//
// opts.Parent is ignored: a task always parents under its config record and the task
// group. The API layer still enforces update access on the parent the caller declared.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	store, ok := s.cfg.Configs.Store(env.Type)
	if !ok {
		return ontology.ID{}, errors.Wrapf(
			validate.ErrValidation, "unknown task type %q", env.Type,
		)
	}
	body, err := imex.Decode[map[string]any](ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	delete(body, "version")
	delete(body, "type")
	delete(body, "name")
	var version imex.Version
	if env.Versioned() {
		version = env.Version
	}
	config, err := store.Normalize(version, msgpack.EncodedJSON(body))
	if err != nil {
		return ontology.ID{}, err
	}
	t := Task{Name: env.Name, Type: env.Type, Config: config}
	if err := s.NewWriter(tx).Create(ctx, &t); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(t.Key), nil
}
