// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/samber/lo"
	dischannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

// Service implements the distribution-layer channel.Retriever interface so it can be
// bound into distribution consumers (chiefly the framer) that need read access to
// channel metadata.
var _ dischannel.Retriever = (*Service)(nil)

// RetrieveByKeys implements dischannel.Retriever, returning the minimal distribution
// representation of the channels matching the provided keys. It returns query.ErrNotFound
// (wrapped) if no channel matches a requested key, mirroring the gorp retrieve semantics
// the framer relied on prior to the retriever hole.
func (s *Service) RetrieveByKeys(ctx context.Context, keys ...Key) ([]dischannel.Channel, error) {
	var chs []Channel
	if err := s.newRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&chs).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return lo.Map(chs, func(c Channel, _ int) dischannel.Channel { return c.Distribution() }), nil
}

// ContainsKeys implements dischannel.Retriever, reporting whether every provided key
// resolves to an existing channel.
func (s *Service) ContainsKeys(ctx context.Context, keys ...Key) (bool, error) {
	return s.newRetrieve().Where(MatchKeys(keys...)).Exists(ctx, nil)
}
