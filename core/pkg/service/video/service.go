// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
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

// ServiceConfig is the configuration for opening a video view service.
type ServiceConfig struct {
	alamos.Instrumentation
	// DB stores the video views.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Signals, when set, broadcasts each dispatch on the sy_video_set channel and each
	// delete on the sy_video_delete channel.
	//
	// [OPTIONAL]
	Signals *signals.Provider
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
	v := validate.New("video")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service creates, retrieves, edits, and deletes video views.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Video]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService opens a video view service. Each configuration overrides the ones before
// it.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
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
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Video]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals != nil {
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:     "sy_video_set",
			Actions:  s.state,
			Resource: ResourceID,
			Table:    tableName,
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Video]{
			Changes:     s.table.Changes,
			Resource:    ResourceID,
			SetDisabled: new(true),
		}); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// Close releases the resources of the service.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// NewWriter opens a writer that creates, edits, and deletes video views in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a query that retrieves video views.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// MoveGroups moves the video views in groups into the project top, in tx, after those
// groups move under top. It returns a validation error when one of the groups holds a
// video view and top is not a project.
func (s *Service) MoveGroups(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
	top *resource.ID,
) error {
	var moved []Video
	if err := s.NewRetrieve().
		Where(MatchGroups(groups...)).
		Entries(&moved).
		Exec(ctx, tx); err != nil || len(moved) == 0 {
		return err
	}
	p, err := project.TopParentProject(top, resource.TypeVideo)
	if err != nil {
		return err
	}
	keys := lo.Map(moved, func(v Video, _ int) Key { return v.Key })
	return s.NewWriter(tx).setProject(ctx, *p, keys...)
}
