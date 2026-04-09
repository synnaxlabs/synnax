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

// Service is the central import/export registry. Each service that supports
// import/export registers its ImporterExporter during layer initialization.
type Service struct {
	db        *gorp.DB
	importers map[ontology.ResourceType]Importer
	exporters map[ontology.ResourceType]Exporter
}

// NewService creates a new import/export registry service.
func NewService(db *gorp.DB) *Service {
	return &Service{
		db:        db,
		importers: make(map[ontology.ResourceType]Importer),
		exporters: make(map[ontology.ResourceType]Exporter),
	}
}

// Register adds an ImporterExporter for the given resource type.
func (s *Service) Register(rt ontology.ResourceType, ie ImporterExporter) {
	s.importers[rt] = ie
	s.exporters[rt] = ie
}

// Import validates and persists the given envelopes within a single transaction.
// Each envelope is routed to the appropriate service by its Type field.
func (s *Service) Import(
	ctx context.Context,
	parent ontology.ID,
	envs []Envelope,
) error {
	return s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for _, env := range envs {
			rt := ontology.ResourceType(env.Type)
			imp, ok := s.importers[rt]
			if !ok {
				return errors.Newf(
					"no importer registered for resource type %q",
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

// Export serializes the requested resources as envelopes.
func (s *Service) Export(
	ctx context.Context,
	resources []ontology.ID,
) ([]Envelope, error) {
	var result []Envelope
	err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for _, id := range resources {
			exp, ok := s.exporters[id.Type]
			if !ok {
				return errors.Newf(
					"no exporter registered for resource type %q",
					id.Type,
				)
			}
			env, err := exp.Export(ctx, tx, id.Key)
			if err != nil {
				return err
			}
			result = append(result, env)
		}
		return nil
	})
	return result, err
}
