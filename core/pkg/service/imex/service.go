// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Service is the central import/export registry. Each service registers its
// ImporterExporter under one or more type strings during layer initialization.
// Type strings are the most specific type identifier (e.g., "log", "modbus_read").
type Service struct {
	db            *gorp.DB
	exportVersion int
	importers     map[string]Importer
	exporters     map[string]Exporter
}

// NewService creates a new import/export registry service. The exportVersion is
// stamped on every exported envelope and is derived from the core version using
// the formula major*1000 + minor.
func NewService(db *gorp.DB, exportVersion int) *Service {
	return &Service{
		db:            db,
		exportVersion: exportVersion,
		importers:     make(map[string]Importer),
		exporters:     make(map[string]Exporter),
	}
}

// Register adds an ImporterExporter for the given type string.
func (s *Service) Register(typeStr string, ie ImporterExporter) {
	s.importers[typeStr] = ie
	s.exporters[typeStr] = ie
}

// RegisterImporter adds an Importer for the given type string.
func (s *Service) RegisterImporter(typeStr string, i Importer) {
	s.importers[typeStr] = i
}

// RegisterExporter adds an Exporter for the given type string.
func (s *Service) RegisterExporter(typeStr string, e Exporter) {
	s.exporters[typeStr] = e
}

// Import validates and persists the given envelopes within a single transaction.
func (s *Service) Import(
	ctx context.Context,
	parent ontology.ID,
	envs []Envelope,
) error {
	return s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for _, env := range envs {
			imp, ok := s.importers[env.Type]
			if !ok {
				return errors.Newf(
					"no importer registered for type %q",
					env.Type,
				)
			}
			if err := imp.Import(ctx, tx, parent, env); err != nil {
				return err
			}
		}
		return nil
	})
}

// Export serializes the requested resources as envelopes. The service stamps
// its exportVersion on every envelope.
func (s *Service) Export(
	ctx context.Context,
	resources []ontology.ID,
) ([]Envelope, error) {
	var result []Envelope
	err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for _, r := range resources {
			exp, ok := s.exporters[string(r.Type)]
			if !ok {
				return errors.Newf(
					"no exporter registered for type %q",
					r.Type,
				)
			}
			env, err := exp.Export(ctx, tx, r.Key)
			if err != nil {
				return err
			}
			env.Version = s.exportVersion
			result = append(result, env)
		}
		return nil
	})
	return result, err
}
