// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	alamos.Instrumentation
	Signals *signals.Provider
	DB      *gorp.DB
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("panel")
	v.NotNil("db", c.DB)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Panel]
	indexes indexes
	state   *actions.State[Key, Action]
}

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		state:   actions.NewState[Key, Action](cfg.DB),
		indexes: newIndexes(),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Panel]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals == nil {
		return s, nil
	}
	// sy_panel_set carries action batches, so the table publishes its deletes only.
	if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
		Name:     "sy_panel_set",
		Actions:  s.state,
		Resource: ResourceID,
		Table:    tableName,
	}); !ok(err, nil) {
		return nil, err
	}
	if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Panel]{
		Changes:     s.table.Changes,
		Resource:    ResourceID,
		SetDisabled: new(true),
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

// CompositionMigrations returns the panel migrations that read other services' staged
// data. Run them once every service table is open.
func (s *Service) CompositionMigrations() []gorp.Migration {
	return versions.CompositionMigrations(s.table)
}

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		svc:        s,
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes: s.indexes,
		baseTX:  s.cfg.DB,
		gorp:    s.table.NewRetrieve(),
	}
}

// MoveGroups moves the panels in groups into the project top, in tx, after those groups
// move under top. It returns a validation error when one of the groups holds a panel
// and top is not a project.
func (s *Service) MoveGroups(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
	top *resource.ID,
) error {
	var moved []Panel
	if err := s.NewRetrieve().
		Where(MatchGroups(groups...)).
		Entries(&moved).
		Exec(ctx, tx); err != nil || len(moved) == 0 {
		return err
	}
	p, err := project.TopParentProject(top, resource.TypePanel)
	if err != nil {
		return err
	}
	keys := lo.Map(moved, func(e Panel, _ int) Key { return e.Key })
	return s.NewWriter(tx).setProject(ctx, *p, keys...)
}
