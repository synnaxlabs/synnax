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
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// ConfigProvider stores strongly typed configurations for a set of task types. A
// config service implements ConfigProvider and registers it via
// Service.RegisterConfigProvider; the task writer then routes config storage for the
// provider's task types through it instead of embedding the config on the task.
// Internal tasks never dispatch to a provider.
//
// All methods receive the task writer's transaction, so a task and its config are
// created, copied, or deleted atomically.
type ConfigProvider interface {
	// Types returns the exact task types this provider handles (e.g.
	// "ni_analog_read"). Type strings must be unique across all registered providers.
	Types() []string
	// Create validates cfg and upserts the typed config record for the given task,
	// returning the ontology ID of the config resource. Create must define the config
	// resource within tx before returning, and must return the same ID for repeated
	// calls with the same task key. It returns a validation error if cfg does not
	// match the provider's schema for taskType.
	Create(
		ctx context.Context,
		tx gorp.Tx,
		task Key,
		taskType string,
		cfg msgpack.EncodedJSON,
	) (ontology.ID, error)
	// Load returns the wire representation of the config stored for the given task.
	// It returns query.ErrNotFound if the provider has no record for the task.
	Load(ctx context.Context, tx gorp.Tx, task Key) (msgpack.EncodedJSON, error)
	// Copy duplicates the config of one task for another, returning the ontology ID
	// of the new config resource. It returns query.ErrNotFound if the source task has
	// no config record.
	Copy(ctx context.Context, tx gorp.Tx, from, to Key) (ontology.ID, error)
	// Delete removes the config record and its ontology resource for the given task.
	// Deleting a task with no config record is a no-op.
	Delete(ctx context.Context, tx gorp.Tx, task Key) error
}

// configProviders is the registry mapping exact task-type strings to providers.
type configProviders struct {
	mu sync.RWMutex
	m  map[string]ConfigProvider
}

func (c *configProviders) register(p ConfigProvider) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string]ConfigProvider)
	}
	for _, t := range p.Types() {
		if _, ok := c.m[t]; ok {
			return errors.Newf(
				"config provider already registered for task type %q",
				t,
			)
		}
	}
	for _, t := range p.Types() {
		c.m[t] = p
	}
	return nil
}

func (c *configProviders) get(taskType string) (ConfigProvider, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.m[taskType]
	return p, ok
}

// RegisterConfigProvider registers a ConfigProvider for its task types. It returns an
// error if another provider is already registered for one of the provider's types.
// Registration must happen during startup, before the service begins handling writes.
func (s *Service) RegisterConfigProvider(p ConfigProvider) error {
	return s.configProviders.register(p)
}

// ResolveConfigs fills the Config field of each task from its registered config
// provider. Tasks without a provider, internal tasks, and tasks whose provider has no
// record yet keep their stored config untouched, so mixed states (configs not yet
// migrated to a provider) resolve correctly.
func (s *Service) ResolveConfigs(ctx context.Context, tx gorp.Tx, tasks []Task) error {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	for i := range tasks {
		t := &tasks[i]
		if t.Internal {
			continue
		}
		p, ok := s.configProviders.get(t.Type)
		if !ok {
			continue
		}
		cfg, err := p.Load(ctx, tx, t.Key)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		t.Config = cfg
	}
	return nil
}

// ConfigDecoder decodes a wire-format config at a single schema version.
type ConfigDecoder[T any] func(cfg msgpack.EncodedJSON) (T, error)

// ErrNoConfigDecoders is returned by DecodeConfig when called with no decoders.
var ErrNoConfigDecoders = errors.New("no config decoders provided")

// DecodeConfig attempts each decoder in order and returns the first successful
// result. Decoders should be ordered newest schema version first, with older versions
// composing their migration into the decoder, so old-shaped wire input normalizes to
// the current shape. If every decoder fails, the first decoder's error is returned,
// since the current version produces the most relevant diagnostics.
func DecodeConfig[T any](
	cfg msgpack.EncodedJSON,
	decoders ...ConfigDecoder[T],
) (T, error) {
	var firstErr error
	for _, decode := range decoders {
		v, err := decode(cfg)
		if err == nil {
			return v, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	var zero T
	if firstErr == nil {
		firstErr = ErrNoConfigDecoders
	}
	return zero, firstErr
}
