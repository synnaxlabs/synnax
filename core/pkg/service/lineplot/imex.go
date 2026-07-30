// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Version is the per-schema version stamped on every exported line plot envelope.
const Version imex.Version = 1

var _ imex.Exporter = (*Service)(nil)

// Export retrieves the line plot identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no line plot exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var lp LinePlot
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&lp).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: lp.Name}
	if err = imex.Encode(&env, lp); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}
