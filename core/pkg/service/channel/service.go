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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures a channel Service.
type ServiceConfig struct {
	// Channel is the distribution-layer channel service.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Status is used to publish error/clear statuses for calculated channels.
	//
	// [REQUIRED]
	Status *status.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("service.channel")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Service is the top-level channel service. It wraps the distribution-layer channel
// service and adds DataType inference for calculated channels on write. The calculated
// channel dependency graph (type repair, status reporting) is a separate reactive
// component opened independently of this Service.
type Service struct{ cfg ServiceConfig }

// NewService opens a channel Service. The ctx is accepted for consistency with other
// service constructors and may be used by future initialization work.
func NewService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// NewArcSymbolResolver returns a resolver that maps cluster channels to Arc symbols by
// name or numeric key, for analyzing and compiling Arc expressions such as calculated
// channels. tx scopes channel lookups; nil consults the service DB directly.
func (s *Service) NewArcSymbolResolver(tx gorp.Tx) arc.SymbolResolver {
	return &symbolResolver{svc: s, tx: tx}
}

// NewWriter returns a Writer that infers DataTypes for calculated channels before
// delegating to the distribution-layer writer.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	w := Writer{writer: s.cfg.Channel.NewWriter(tx)}
	w.analyzer = NewAnalyzer(s.NewArcSymbolResolver(tx))
	return w
}

// NewRetrieve opens a query to retrieve channels from the cluster.
func (s *Service) NewRetrieve() Retrieve { return s.cfg.Channel.NewRetrieve() }

// Group returns the ontology group that channels are created under.
func (s *Service) Group() group.Group { return s.cfg.Channel.Group() }

// Observe returns an observable that notifies callers of changes to channel entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Channel]] {
	return s.cfg.Channel.Observe()
}

// Create creates a single channel, inferring the DataType for calculated channels.
func (s *Service) Create(ctx context.Context, ch *Channel, opts ...CreateOption) error {
	return s.NewWriter(nil).Create(ctx, ch, opts...)
}

// CreateMany creates multiple channels, inferring DataTypes for calculated channels.
func (s *Service) CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error {
	return s.NewWriter(nil).CreateMany(ctx, channels, opts...)
}

// Delete deletes a channel by key.
func (s *Service) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return s.NewWriter(nil).Delete(ctx, key, allowInternal)
}

// DeleteMany deletes multiple channels by key.
func (s *Service) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return s.NewWriter(nil).DeleteMany(ctx, keys, allowInternal)
}

// DeleteByName deletes a channel by name.
func (s *Service) DeleteByName(ctx context.Context, name string, allowInternal bool) error {
	return s.NewWriter(nil).DeleteByName(ctx, name, allowInternal)
}

// DeleteManyByNames deletes multiple channels by name.
func (s *Service) DeleteManyByNames(ctx context.Context, names []string, allowInternal bool) error {
	return s.NewWriter(nil).DeleteManyByNames(ctx, names, allowInternal)
}

// Rename renames a channel.
func (s *Service) Rename(ctx context.Context, key Key, newName string, allowInternal bool) error {
	return s.NewWriter(nil).Rename(ctx, key, newName, allowInternal)
}

// RenameMany renames multiple channels.
func (s *Service) RenameMany(ctx context.Context, keys []Key, names []string, allowInternal bool) error {
	return s.NewWriter(nil).RenameMany(ctx, keys, names, allowInternal)
}

// MapRename renames channels using an old-name to new-name mapping.
func (s *Service) MapRename(ctx context.Context, names map[string]string, allowInternal bool) error {
	return s.NewWriter(nil).MapRename(ctx, names, allowInternal)
}

// CountExternalNonVirtual returns the number of external non-virtual channels in the
// service.
func (s *Service) CountExternalNonVirtual() uint32 {
	return s.cfg.Channel.CountExternalNonVirtual()
}
