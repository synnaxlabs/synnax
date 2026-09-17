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
	"maps"
	"slices"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Service is the central import/export registry. Handlers are registered via
// [Service.RegisterImportExporter], [Service.RegisterImporter], or
// [Service.RegisterExporter], and [Service.Import] and [Service.Export] route to them
// by type. Service is safe for concurrent registration and lookup.
type Service struct {
	mu        sync.RWMutex
	importers map[string]Importer
	exporters map[ontology.ResourceType]Exporter
}

// NewService creates a new, empty [Service], safe for concurrent use.
func NewService() *Service {
	return &Service{
		importers: make(map[string]Importer),
		exporters: make(map[ontology.ResourceType]Exporter),
	}
}

// RegisterImportExporter registers a single handler for both halves under its own type.
// Use this for symmetric services (one resource type for both import and export) —
// e.g., logs, schematics. It returns ErrTypeTaken if either half of the type is already
// registered, leaving both halves untouched.
func (s *Service) RegisterImportExporter(ie ImportExporter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := ie.Type()
	if _, taken := s.importers[string(t)]; taken {
		return typeTakenError("importer", string(t))
	}
	if _, taken := s.exporters[t]; taken {
		return typeTakenError("exporter", string(t))
	}
	s.importers[string(t)] = ie
	s.exporters[t] = ie
	return nil
}

// RegisterImporter adds an Importer for t. Use it when a service registers under
// fine-grained type strings ("http_read") but exports under one coarse type ("task").
// It returns ErrTypeTaken if an Importer is already registered for t.
func (s *Service) RegisterImporter(t string, i Importer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.importers[t]; taken {
		return typeTakenError("importer", t)
	}
	s.importers[t] = i
	return nil
}

// RegisterExporter adds an [Exporter] for the given resource type. See
// [Service.RegisterImporter] for the asymmetric-registration use case. It returns
// ErrTypeTaken if an Exporter is already registered for the type.
func (s *Service) RegisterExporter(e Exporter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.exporters[e.Type()]; taken {
		return typeTakenError("exporter", string(e.Type()))
	}
	s.exporters[e.Type()] = e
	return nil
}

// ErrTypeTaken is returned when a handler is registered for a type another handler
// already claims. Two services claiming one type is a wiring mistake: the second
// registration would silently shadow the first.
var ErrTypeTaken = errors.New("import/export type already registered")

func typeTakenError(kind, typ string) error {
	return errors.Wrapf(ErrTypeTaken, "%s for type %q", kind, typ)
}

// ImporterType returns the ontology resource type created by the Importer registered
// under t. It returns a validation error scoped to the "type" field when no importer is
// registered for t.
func (s *Service) ImporterType(t string) (ontology.ResourceType, error) {
	s.mu.RLock()
	imp, ok := s.importers[t]
	s.mu.RUnlock()
	if !ok {
		return "", notFoundError(t, "importer")
	}
	return imp.Type(), nil
}

func notFoundError[T ~string](typ T, kind string) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "no %s registered for type %q", kind, typ),
		"type",
	)
}

// ExporterRegistered reports whether an Exporter is registered for t.
func (s *Service) ExporterRegistered(t ontology.ResourceType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.exporters[t]
	return ok
}

// ResolveType returns the registration type string Import will route envelope to. A
// non-empty envelope.Type passes through; a typeless envelope's body is offered to
// every Importer's Match in sorted type order, first claim winning. It returns a
// validation error scoped to the "type" field when nothing claims the body.
func (s *Service) ResolveType(envelope Envelope) (string, error) {
	if envelope.Type != "" {
		return envelope.Type, nil
	}
	body := envelope.body
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range slices.Sorted(maps.Keys(s.importers)) {
		if s.importers[t].Match(body) {
			return t, nil
		}
	}
	return "", newFieldError("type", "file does not match any known resource format")
}

// Import routes envelope to its Importer, persists it on tx, and returns the created
// resource's ID. The opts.FileName fallback is applied before routing; opts then
// reaches the importer untouched. It returns a validation error scoped to "parent" when
// opts.Parent is zero, to "type" when nothing is registered, and to "name" when the
// envelope is still nameless after the fallback.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	envelope Envelope,
	opts ImportOptions,
) (ontology.ID, error) {
	if opts.Parent.IsZero() {
		return ontology.ID{}, validate.PathedError(validate.ErrRequired, "parent")
	}
	if envelope.Type == "" {
		typ, err := s.ResolveType(envelope)
		if err != nil {
			return ontology.ID{}, err
		}
		envelope.Type = typ
	}
	s.mu.RLock()
	importer, ok := s.importers[envelope.Type]
	s.mu.RUnlock()
	if !ok {
		return ontology.ID{}, notFoundError(envelope.Type, "importer")
	}
	if envelope.Name == "" {
		envelope.Name = filename.Stem(opts.FileName)
	}
	if envelope.Name == "" {
		return ontology.ID{}, newFieldError("name", "name must be a non-empty string")
	}
	id, err := importer.Import(ctx, tx, envelope, opts)
	if err != nil {
		return ontology.ID{}, errors.Wrap(err, "import envelope")
	}
	return id, nil
}

// Export routes resource to its Exporter and returns the resulting envelope. It returns
// a validation error scoped to the "type" field when no Exporter is registered.
func (s *Service) Export(
	ctx context.Context,
	resource ontology.ID,
) (Envelope, error) {
	s.mu.RLock()
	exporter, ok := s.exporters[resource.Type]
	s.mu.RUnlock()
	if !ok {
		return Envelope{}, notFoundError(resource.Type, "exporter")
	}
	env, err := exporter.Export(ctx, resource)
	if err != nil {
		return Envelope{}, errors.Wrap(err, "export resource")
	}
	return env, nil
}
