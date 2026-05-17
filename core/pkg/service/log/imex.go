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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations"
	"github.com/synnaxlabs/x/gorp"
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
) (string, error) {
	l, err := migrateData(env.Version, env.Data)
	if err != nil {
		return "", err
	}
	l.Name = env.Name
	if err := s.NewWriter(tx).Create(ctx, uuid.Nil, &l); err != nil {
		return "", err
	}
	return l.Key.String(), nil
}

func (s *Service) Export(ctx context.Context, key string) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&l).Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	return imex.Envelope{
		Version: migrations.LatestVersion,
		Type:    string(ontology.ResourceTypeLog),
		Name:    l.Name,
		Data:    l.data(),
	}, nil
}

func (l Log) data() map[string]any {
	return map[string]any{
		"channels":               l.Channels,
		"remote_created":         l.RemoteCreated,
		"timestamp_precision":    l.TimestampPrecision,
		"show_channel_names":     l.ShowChannelNames,
		"show_receipt_timestamp": l.ShowReceiptTimestamp,
	}
}
