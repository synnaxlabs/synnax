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
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	Key          = distchannel.Key
	Keys         = distchannel.Keys
	LocalKey     = distchannel.LocalKey
	Channel      = distchannel.Channel
	Operation    = distchannel.Operation
	CreateOption = distchannel.CreateOption
)

var (
	RetrieveIfNameExists                        = distchannel.RetrieveIfNameExists
	OverwriteIfNameExistsAndDifferentProperties = distchannel.OverwriteIfNameExistsAndDifferentProperties
	CreateWithoutGroupRelationship              = distchannel.CreateWithoutGroupRelationship
	ParseKey                                    = distchannel.ParseKey
	OntologyID                                  = distchannel.OntologyID
	MatchKeys                                   = distchannel.MatchKeys
	MatchNames                                  = distchannel.MatchNames
	OntologyIDsFromChannels                     = distchannel.OntologyIDsFromChannels
	KeysFromChannels                            = distchannel.KeysFromChannels
	MatchLeaseholders                           = distchannel.MatchLeaseholders
	MatchDataTypes                              = distchannel.MatchDataTypes
	MatchVirtual                                = distchannel.MatchVirtual
	MatchIsIndex                                = distchannel.MatchIsIndex
	MatchInternal                               = distchannel.MatchInternal
	Not                                         = distchannel.Not
	NewRandomName                               = distchannel.NewRandomName
)

// ServiceConfig configures a channel Service.
type ServiceConfig struct {
	// DB is the underlying database for transactional operations.
	DB *gorp.DB
	// Distribution is the distribution-layer channel service.
	Distribution *distchannel.Service
	// Status is used to publish error/clear statuses for calculated channels.
	Status *status.Service
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (c ServiceConfig) Validate() error {
	v := validate.New("service.channel")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "distribution", c.Distribution)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Service is the top-level channel service. It wraps the distribution-layer
// channel service and adds DataType inference for calculated channels on write.
// The calculated channel dependency graph (type repair, status reporting) is a
// separate reactive component opened independently of this Service.
type Service struct {
	*distchannel.Service
	cfg ServiceConfig
}

// OpenService opens a channel Service. The ctx is accepted for consistency with
// other service constructors and may be used by future initialization work.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{Service: cfg.Distribution, cfg: cfg}, nil
}

// Wrap creates a Service that delegates directly to the distribution-layer channel
// service without calculated channel features (type inference, dependency tracking).
// Use OpenService for full functionality.
func Wrap(dist *distchannel.Service) *Service {
	return &Service{Service: dist, cfg: ServiceConfig{Distribution: dist}}
}

// Close is a no-op that intentionally shadows the embedded distribution-layer
// Service.Close. Closing the service-layer Service must not tear down the shared
// distribution channel service, which is owned and closed by the distribution
// layer.
func (s *Service) Close() error { return nil }

// NewArcSymbolResolver returns a resolver that maps cluster channels to Arc
// symbols by name or numeric key, for analyzing and compiling Arc expressions
// such as calculated channels. tx scopes channel lookups; nil consults the
// service DB directly.
func (s *Service) NewArcSymbolResolver(tx gorp.Tx) arc.SymbolResolver {
	return &channelResolver{dist: s.cfg.Distribution, tx: tx}
}

// NewWriter returns a Writer that infers DataTypes for calculated channels
// before delegating to the distribution-layer writer.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	w := Writer{Writer: s.cfg.Distribution.NewWriter(tx)}
	w.tx = gorp.OverrideTx(s.cfg.DB, tx)
	w.analyzer = analyzer.New(s.NewArcSymbolResolver(tx))
	return w
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
