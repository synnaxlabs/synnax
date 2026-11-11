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
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.Exporter = (*Service)(nil)

// Type implements imex.Exporter.
func (s *Service) Type() resource.Type { return resource.TypeTask }

// Export retrieves the task identified by id from snap and serializes it as an
// imex.Envelope. The task's type-specific config is merged flat into the envelope body
// — the driver reads a task file directly as its config, so the config is not nested
// under a `config` field; version, the fine-grained task type (e.g. "opc_read"), and
// name are stamped on top. The version is the config type's own number line, taken from
// its config store. Routing to this exporter still happens under the coarse "task"
// resource type via Type. It returns query.ErrNotFound if no task exists for id.Key,
// and a validation error if the task's type has no file form.
func (s *Service) Export(
	ctx context.Context,
	snap gorp.Snapshot,
	id resource.ID,
) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var t Task
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&t).
		Exec(ctx, snap); err != nil {
		return imex.Envelope{}, err
	}
	if slices.Contains(s.cfg.ImExExcluded, t.Type) {
		return imex.Envelope{}, errors.Wrapf(
			validate.ErrValidation, "tasks of type %q have no file form", t.Type,
		)
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
	if err = env.Encode(body); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

var _ imex.Importer = (*Service)(nil)

// retiredTypes are task types released builds created that no longer exist. An import
// of one fails with a retirement message instead of "unknown task type".
var retiredTypes = set.New("sequence", "heartbeat", "opcScanner", "Rack State")

// Match always reports false: task files always carry a type header, so a typeless
// body is never a task.
func (*Service) Match(map[string]any) bool { return false }

// Decode decodes env as a task file — the flat body is the task's config with the type,
// name, and version headers stamped on top — that its Write creates as a draft on no
// rack. The version dispatches through the type's config store: legacy versions
// upgrade through the same rewrites as the boot migration, and a version newer than
// the store's returns a validation error. A file with no version header is a legacy
// Console export at version zero.
//
// The Write ignores opts: a task always parents under its config record and the task
// group.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	store, ok := s.cfg.Configs.Store(env.Type)
	if !ok {
		if retiredTypes.Contains(env.Type) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"task type %q was removed and can no longer be imported", env.Type,
			)
		}
		return nil, errors.Wrapf(
			validate.ErrValidation, "unknown task type %q", env.Type,
		)
	}
	body, err := env.Decode[map[string]any](ctx)
	if err != nil {
		return nil, err
	}
	delete(body, "version")
	delete(body, "type")
	delete(body, "name")
	decoded, err := store.Decode(ctx, env.Version, msgpack.EncodedJSON(body))
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		_ imex.ImportOptions,
	) (resource.ID, error) {
		config, err := decoded.Normalize(ctx, tx)
		if err != nil {
			return resource.ID{}, err
		}
		t := Task{Name: env.Name, Type: env.Type, Config: config}
		if err := s.NewWriter(tx).Create(ctx, &t); err != nil {
			return resource.ID{}, err
		}
		return ResourceID(t.Key), nil
	}, nil
}
