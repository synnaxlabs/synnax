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
