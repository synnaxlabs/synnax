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
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Version is the per-schema version stamped on every exported task envelope.
const Version imex.Version = 1

var _ imex.Exporter = (*Service)(nil)

// Export retrieves the task identified by id and serializes it as an imex.Envelope. The
// task's type-specific config is merged flat into the envelope body — the driver reads
// a task file directly as its config, so the config is not nested under a `config`
// field; version, the fine-grained task type (e.g. "opc_read"), and name are stamped on
// top. Routing to this exporter still happens under the coarse "task" ontology type via
// Type. Flattening lives here rather than in imex because it is temporary: once task
// configs are strongly typed, Export will encode the typed struct like every other
// resource. It returns query.ErrNotFound if no task exists for id.Key.
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
	body := map[string]any(t.Config)
	if body == nil {
		body = make(map[string]any)
	}
	body["type"] = t.Type
	body["name"] = t.Name
	env := imex.Envelope{Version: Version}
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
// no rack. Legacy Console exports upgrade through the same transforms as the boot
// migration. It returns a validation error for a file version newer than Version.
//
// opts.Parent is ignored: a task always parents under its config record and the task
// group. The API layer still enforces update access on the parent the caller declared.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	if env.Versioned() && env.Version > Version {
		return ontology.ID{}, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation, "unsupported task file version %d", env.Version,
			),
			"version",
		)
	}
	body, err := imex.Decode[map[string]any](ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	delete(body, "version")
	delete(body, "type")
	delete(body, "name")
	t := Task{
		Name:   env.Name,
		Type:   env.Type,
		Config: v3.Transform(env.Type, body),
	}
	if err := s.NewWriter(tx).Create(ctx, &t); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(t.Key), nil
}
