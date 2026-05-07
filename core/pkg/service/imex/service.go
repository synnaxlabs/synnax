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
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Service is the central import/export registry. Each service registers its
// ImporterExporter under one or more type strings during layer initialization.
// Type strings are the most specific type identifier (e.g., "log", "modbus_read").
type Service struct {
	db        *gorp.DB
	importers map[string]Importer
	exporters map[string]Exporter
}

// NewService creates a new import/export registry service. Each registered
// handler is responsible for stamping its own per-schema version on the
// envelopes it produces; the registry does not impose a centralized version.
func NewService(db *gorp.DB) *Service {
	return &Service{
		db:        db,
		importers: make(map[string]Importer),
		exporters: make(map[string]Exporter),
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

// Import validates and persists the given envelopes within a single
// transaction, returning the newly-assigned key for each resource in the same
// order as envs. The envelope's key is ignored; each handler always assigns a
// fresh key.
func (s *Service) Import(
	ctx context.Context,
	envs []Envelope,
) ([]string, error) {
	keys := make([]string, len(envs))
	err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, env := range envs {
			imp, ok := s.importers[env.Type]
			if !ok {
				return errors.Newf(
					"no importer registered for type %q",
					env.Type,
				)
			}
			payload, err := decodeImportPayload(env)
			if err != nil {
				return err
			}
			key, err := imp.Import(ctx, tx, payload)
			if err != nil {
				return err
			}
			keys[i] = key
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// decodeImportPayload decodes the envelope's Data into a generic
// map[string]any so the handler can route on Version and dispatch to the right
// schema parser. The envelope's promoted Key is intentionally dropped.
func decodeImportPayload(env Envelope) (ImportPayload, error) {
	payload := ImportPayload{Version: env.Version, Name: env.Name}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &payload.Data); err != nil {
			return ImportPayload{}, errors.Wrap(err, "envelope data must be a JSON object")
		}
	}
	return payload, nil
}

// Export serializes the requested resources as envelopes. Each registered
// handler stamps its own per-schema version on the envelopes it returns.
func (s *Service) Export(
	ctx context.Context,
	resources []ontology.ID,
) ([]Envelope, error) {
	var result []Envelope
	for _, r := range resources {
		exp, ok := s.exporters[string(r.Type)]
		if !ok {
			return nil, errors.Newf(
				"no exporter registered for type %q",
				r.Type,
			)
		}
		env, err := exp.Export(ctx, r.Key)
		if err != nil {
			return nil, err
		}
		result = append(result, env)
	}
	return result, nil
}
