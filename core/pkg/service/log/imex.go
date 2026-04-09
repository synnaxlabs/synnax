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
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

type importExporter struct{ svc *Service }

// ImportExporter returns an imex.ImporterExporter for the log service.
func (s *Service) ImportExporter() imex.ImporterExporter {
	return &importExporter{svc: s}
}

func (ie *importExporter) Import(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	env imex.Envelope,
) error {
	data, err := ie.importData(env.Version, env.Data)
	if err != nil {
		return err
	}
	key, err := uuid.Parse(env.Key)
	if err != nil {
		return err
	}
	wsKey, err := uuid.Parse(parent.Key)
	if err != nil {
		return err
	}
	l := Log{Key: key, Name: env.Name, Data: data}
	return ie.svc.NewWriter(tx).Create(ctx, wsKey, &l)
}

func (ie *importExporter) importData(
	version string,
	data map[string]any,
) (map[string]any, error) {
	switch version {
	case v1.Version:
		var d v1.Data
		if err := v1.Schema.Parse(data, &d); err != nil {
			return nil, err
		}
		dumped, err := v1.Schema.Dump(d)
		if err != nil {
			return nil, err
		}
		return dumped.(map[string]any), nil
	case v0.Version:
		var d v0.Data
		if err := v0.Schema.Parse(data, &d); err != nil {
			return nil, err
		}
		migrated, err := v1.Migrate(d)
		if err != nil {
			return nil, err
		}
		dumped, err := v1.Schema.Dump(migrated)
		if err != nil {
			return nil, err
		}
		return dumped.(map[string]any), nil
	default:
		return nil, errors.Newf("unknown log data version %q", version)
	}
}

func (ie *importExporter) Export(
	ctx context.Context,
	tx gorp.Tx,
	key string,
) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err := ie.svc.NewRetrieve().WhereKeys(k).Entry(&l).Exec(ctx, tx); err != nil {
		return imex.Envelope{}, err
	}
	return imex.Envelope{
		Version: v1.Version,
		Type:    string(ontology.ResourceTypeLog),
		Key:     l.Key.String(),
		Name:    l.Name,
		Data:    l.Data,
	}, nil
}
