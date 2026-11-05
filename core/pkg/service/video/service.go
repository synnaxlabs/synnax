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
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	// Ontology relates video views to other resources.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
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
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("video")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
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
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = cfg.Signals.PublishActions(
			ctx,
			signals.ActionPublisherConfig[Key, Action]{
				Observable: s.state,
				Name:       "video",
			},
		); !ok(err, sig) {
			return nil, err
		}
		deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
		deleteCfg.DisableSet = true
		if sig, err = cfg.Signals.PublishFromGorp(ctx, deleteCfg); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close releases the resources of the service.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction subscribes handler to the actions that Dispatch applies. The handler runs
// inside Dispatch after its transaction commits.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Dispatch applies acts atomically to the video view with the given key. Dispatches to
// one view run one at a time, each in a transaction that commits before its actions
// are broadcast. dispatchKey goes onto the broadcast so the sender recognizes its own
// echo. Returns a validation error, and stores nothing, when the actions leave the
// view invalid.
func (s *Service) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) error {
	return s.state.Dispatch(
		ctx,
		key,
		dispatchKey,
		acts,
		func(tx gorp.Tx) ([]Action, error) {
			return nil, s.table.NewUpdate().Where(gorp.MatchKeys[Key, Video](key)).
				ChangeErr(func(_ gorp.Context, v Video) (Video, error) {
					reduced, err := Reduce(v, acts...)
					if err != nil {
						return Video{}, err
					}
					if err = reduced.Validate(); err != nil {
						return Video{}, err
					}
					return reduced, nil
				}).Exec(ctx, tx)
		},
	)
}

// NewWriter opens a writer that runs in tx, or directly on the database when tx is
// nil.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a query that retrieves video views.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}
