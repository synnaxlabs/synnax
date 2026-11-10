// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias/versions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ParentRetriever is an interface for retrieving the parent range key for a
// given range. This allows the alias service to implement inheritance without
// a direct dependency on the ranger service implementation.
type ParentRetriever interface {
	RetrieveParentKey(
		ctx context.Context,
		key ranger.Key,
		snap gorp.Snapshot,
	) (ranger.Key, error)
}

// ServiceConfig is the configuration for opening the alias.Service.
type ServiceConfig struct {
	DB              *gorp.DB
	Channel         *channel.Service
	Signals         *signals.Provider
	ParentRetriever ParentRetriever
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.ranger.alias")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("parent_retriever", c.ParentRetriever)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ParentRetriever = override.Nil(c.ParentRetriever, other.ParentRetriever)
	return c
}

// Service is the main entry point for managing channel aliases on ranges.
type Service struct {
	closer  xio.MultiCloser
	table   *gorp.Table[string, Alias]
	search  *gorp.SearchIndex[string, Alias]
	indexes indexes
	cfg     ServiceConfig
	state   *actions.State[string, Action]
}

// OpenService opens a new alias.Service with the provided configuration.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		search:  newSearchIndex(),
		indexes: newIndexes(),
		state:   actions.NewState[string, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[string, Alias]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         append(s.indexes.all(), s.search),
		References:      s.indexes.references(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals != nil {
		if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[string, Alias]{
			Changes:    s.table.Changes,
			Resource:   resourceID,
			SetKey:     signals.UnmarshalEntryKey[string, Alias],
			SetName:    "sy_range_alias_set",
			DeleteName: "sy_range_alias_delete",
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[string, Action]{
			Name:     "sy_range_alias_action_set",
			Actions:  s.state,
			Resource: resourceID,
		}); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the service and releases any resources.
func (s *Service) Close() error { return s.closer.Close() }

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[string, Action] { return s.state }

// NewWriter opens a new Writer to create, change, and delete aliases.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		channel:    s.cfg.Channel,
		table:      s.table,
		indexes:    s.indexes,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewReader opens a new Reader to retrieve aliases.
func (s *Service) NewReader(snap gorp.Snapshot) Reader {
	return Reader{
		snap:            snap,
		parentRetriever: s.cfg.ParentRetriever,
		table:           s.table,
		indexes:         s.indexes,
	}
}
