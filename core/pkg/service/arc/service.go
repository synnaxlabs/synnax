// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/ranges"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Arc service.
type ServiceConfig struct {
	// DB is the database that the Arc service will store Arcs in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Table stores the Arcs. The caller opens it, and closes it after the Service.
	//
	// [REQUIRED]
	Table *Table
	// Channel is used for retrieving channel information from the cluster.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Task is used for creating and deleting the tasks that deploy Arcs to racks.
	//
	// [REQUIRED]
	Task *task.Service
	// MatchTasks returns a filter for the tasks that run any of the Arcs with keys.
	//
	// [REQUIRED]
	MatchTasks func(arcs ...Key) task.Filter
	// Status is used to check whether an Arc's task is running before undeploying it.
	//
	// [REQUIRED]
	Status *status.Service
	// Signals is used to broadcast collaborative-edit actions to the cluster.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	// ImEx is the import/export registry the Arc service registers itself with as the
	// exporter for Arc resources during OpenService.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// TextSweepQuiescence is how long an Arc's text must go unedited before its
	// tombstoned characters become eligible to be reclaimed.
	//
	// [OPTIONAL] - Defaults to 5 seconds
	TextSweepQuiescence telem.TimeSpan
	// TextSweepThreshold is the number of tombstoned characters that must accumulate
	// before a sweep is broadcast.
	//
	// [OPTIONAL] - Defaults to 128.
	TextSweepThreshold int
	// TaskSyncDebounce is the trailing delay between an edit and the rewrite of the
	// Arc's task config, so a burst of edits produces one rewrite.
	//
	// [OPTIONAL] - Defaults to 500 milliseconds.
	TaskSyncDebounce telem.TimeSpan
	// Now returns the current cluster time. It gates the text sweeper's quiescence
	// check and is injectable for testing.
	//
	// [OPTIONAL] - Defaults to telem.Now.
	Now func() telem.TimeStamp
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Table = override.Nil(c.Table, other.Table)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Task = override.Nil(c.Task, other.Task)
	c.MatchTasks = override.Nil(c.MatchTasks, other.MatchTasks)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.TextSweepQuiescence = override.Numeric(
		c.TextSweepQuiescence,
		other.TextSweepQuiescence,
	)
	c.TextSweepThreshold = override.Numeric(
		c.TextSweepThreshold,
		other.TextSweepThreshold,
	)
	c.TaskSyncDebounce = override.Numeric(c.TaskSyncDebounce, other.TaskSyncDebounce)
	c.Now = override.Nil(c.Now, other.Now)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("arc")
	v.NotNil("db", c.DB)
	v.NotNil("table", c.Table)
	v.NotNil("channel", c.Channel)
	v.NotNil("task", c.Task)
	v.NotNil("match_tasks", c.MatchTasks)
	v.NotNil("status", c.Status)
	v.NotNil("imex", c.ImEx)
	v.Positive("task_sync_debounce", c.TaskSyncDebounce)
	return v.Error()
}

// Service is the primary service for retrieving and modifying arcs from Synnax.
type Service struct {
	table   *gorp.Table[Key, Arc]
	indexes indexes
	closer  xio.MultiCloser
	cfg     ServiceConfig
	// directories stores the directories below root.
	directories *directory.Table
	// root is the directory that holds every Arc.
	root directory.Key
	// reserved holds the standard module names, which no entry of root may take.
	reserved set.Set[string]
	state    *actions.State[Key, Action]
	sweeper  textSweeper
	taskSync *debounce.Keyed[Key]
	// resyncs and deletes hold the Arcs that a transaction dispatched to and deleted.
	// followCommit reads them once the transaction commits.
	resyncs *gorp.Attachment[Key]
	deletes *gorp.Attachment[Key]
}

// NewRoot builds the production analysis root: the STL, status, and ranges modules, the
// Core channel resolver, and the user modules below Arcs. The root looks up channels
// and modules in snap and records each lookup in lookups. A root resolves each channel
// name and module the same way for its lifetime, so build one per compile.
func (s *Service) NewRoot(snap gorp.Snapshot, lookups *channel.Lookups) *symbol.Symbol {
	return s.newRoot(
		s.cfg.Channel.NewCheckedResolver(snap, lookups),
		s.checkedModuleSource(snap, lookups),
	)
}

func (s *Service) newRoot(
	channels symbol.Resolver,
	source func(context.Context, string) (arctext.Text, error),
) *symbol.Symbol {
	modules := arctext.NewLoader(arctext.LoaderConfig{
		Source:   source,
		Channels: channels,
		Ambient:  standardSymbols,
		Config:   parser.Config{AllowDashedNames: s.AllowDashedNames()},
	})
	return symbol.NewRoot(channels, standardSymbols(), symbol.WithModules(modules))
}

func standardSymbols() []*symbol.Symbol {
	return slices.Concat(stl.NewSymbols(), arcstatus.NewSymbols(), ranges.NewSymbols())
}

// LSPConfig configures the language server of one editor session.
type LSPConfig struct {
	// Allowed reads the access of the subject of the session and returns whether the
	// subject can retrieve a record. The server calls it once for each analysis.
	Allowed func(context.Context) (func(resource.ID) bool, error)
}

// NewLSP opens the language server of one editor session. The server treats the
// channels and modules that the subject cannot retrieve as unknown names.
func (s *Service) NewLSP(cfg LSPConfig) (*lsp.Server, error) {
	return lsp.New(lsp.Config{
		Instrumentation:  s.cfg.Child("lsp"),
		NewRoot:          func() *symbol.Symbol { return s.newLSPRoot(cfg.Allowed) },
		AllowDashedNames: s.AllowDashedNames(),
		OnRename: func(
			ctx context.Context,
			sym *symbol.Symbol,
			oldName,
			newName string,
		) error {
			if sym.Kind != symbol.KindChannel {
				return nil
			}
			return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return s.cfg.Channel.NewWriter(tx).
					Rename(ctx, channel.Key(sym.ID), newName, false)
			})
		},
		OnExternalChange: observe.Translator[gorp.TxReader[channel.Key, channel.Channel], struct{}]{
			Observable: s.cfg.Channel.Observe(),
			Translate: func(
				context.Context,
				gorp.TxReader[channel.Key, channel.Channel],
			) (struct{}, bool) {
				return struct{}{}, true
			},
		},
	})
}

// newLSPRoot builds a root that reads the DB with no snapshot, since the LSP analyzes
// outside any scope the Core holds. It hides the records that allowed rejects.
func (s *Service) newLSPRoot(
	allowed func(context.Context) (func(resource.ID) bool, error),
) *symbol.Symbol {
	access := &readAccess{read: allowed}
	return s.newRoot(
		channel.NewConsistentResolver(s.cfg.Channel.NewArcSymbolResolver(
			s.cfg.DB,
			channel.Match(func(
				ctx gorp.Context,
				_ channel.Retrieve,
				ch *channel.Channel,
			) (bool, error) {
				return access.allows(ctx, ch.ResourceID())
			}),
		)),
		s.moduleSource(s.cfg.DB, Match(func(
			ctx gorp.Context,
			_ Retrieve,
			a *Arc,
		) (bool, error) {
			return access.allows(ctx, a.ResourceID())
		})),
	)
}

// readAccess reads the access of an LSP subject at the first lookup of one root.
type readAccess struct {
	read    func(context.Context) (func(resource.ID) bool, error)
	once    sync.Once
	allowed func(resource.ID) bool
	err     error
}

func (a *readAccess) allows(ctx context.Context, id resource.ID) (bool, error) {
	a.once.Do(func() { a.allowed, a.err = a.read(ctx) })
	if a.err != nil {
		return false, a.err
	}
	return a.allowed(id), nil
}

func (s *Service) Close() error { return s.closer.Close() }

// NewDirectoryRetrieve opens a query for Arc directories.
func (s *Service) NewDirectoryRetrieve() directory.Retrieve {
	return s.directories.NewRetrieve()
}

// AllowDashedNames reports whether Arc may treat '-' as an identifier character, which
// is permitted exactly when channel-name validation is disabled.
func (s *Service) AllowDashedNames() bool { return !s.cfg.Channel.ShouldValidateNames() }

// CompileProgram retrieves the automation with key and compiles its program. A module
// is a validation error.
func (s *Service) CompileProgram(ctx context.Context, key Key) (AutomationArc, error) {
	var entry Arc
	err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.NewRetrieve().Where(MatchKeys(key)).Entry(&entry).Exec(ctx, snap)
	})
	if err != nil {
		return AutomationArc{}, err
	}
	automation, ok := entry.Variant.(AutomationArc)
	if !ok {
		return AutomationArc{}, errors.Wrapf(
			validate.ErrValidation,
			"modules cannot be deployed independently. Import %q from an "+
				"automation instead",
			entry.Base().Name,
		)
	}
	prog, err := channel.CompileConsistent(ctx, s.cfg.DB, func(
		snap gorp.Snapshot,
		lookups *channel.Lookups,
	) (arc.Program, error) {
		root := s.NewRoot(snap, lookups)
		switch src := automation.Source.Variant.(type) {
		case TextSource:
			return arc.CompileText(ctx, src.Text.Materialize(), root,
				arc.WithAllowDashedNames(s.AllowDashedNames()))
		case GraphSource:
			return arc.CompileGraph(ctx, src.Graph, root,
				arc.WithAllowDashedNames(s.AllowDashedNames()))
		}
		return arc.Program{}, nil
	})
	if err != nil {
		return AutomationArc{}, err
	}
	automation.Program = &prog
	return automation, nil
}

// OpenService instantiates a new Arc service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the ConfigValues struct for information on which fields should be set.
func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{
		TextSweepQuiescence: 5 * telem.Second,
		TextSweepThreshold:  128,
		TaskSyncDebounce:    500 * telem.Millisecond,
		Now:                 telem.Now,
	}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:         cfg,
		reserved:    reservedNames(),
		state:       actions.NewState[Key, Action](cfg.DB),
		table:       cfg.Table.table,
		indexes:     cfg.Table.indexes,
		directories: cfg.Table.directories,
		root:        cfg.Table.root,
		resyncs:     gorp.NewAttachment[Key](),
		deletes:     gorp.NewAttachment[Key](),
		sweeper: newTextSweeper(
			cfg.Now,
			cfg.TextSweepQuiescence,
			cfg.TextSweepThreshold,
		),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.taskSync, err = debounce.NewKeyed(debounce.KeyedConfig[Key]{
		Delay:    cfg.TaskSyncDebounce.Duration(),
		MaxDelay: cfg.TaskSyncDebounce.Duration() * 4,
		Callback: s.resyncTask,
	}); !ok(err, s.taskSync) {
		return nil, err
	}
	ok(nil, xio.NoFailCloserFunc(cfg.DB.Observe().OnChange(s.followCommit)))
	if err = cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	for _, t := range []Type{TypeAutomation, TypeModule} {
		if err = cfg.ImEx.RegisterImporter(versions.FileType(t), s); !ok(err, nil) {
			return nil, err
		}
	}
	if cfg.Signals != nil {
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:     "sy_arc_set",
			Actions:  s.state,
			Resource: ResourceID,
			Table:    tableName,
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Arc]{
			Changes:     s.table.Changes,
			Resource:    ResourceID,
			SetDisabled: new(true),
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishTable(
			ctx,
			signals.TableConfig[directory.Key, directory.Directory]{
				Changes:    s.directories.Changes,
				Resource:   directory.ResourceID,
				SetName:    directory.SetChannelName,
				DeleteName: directory.DeleteChannelName,
			},
		); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// NewWriter opens a new writer for creating, updating, and deleting Arcs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:          tx,
		root:        s.root,
		directories: s.directories,
		reserved:    s.reserved,
		tasks:       s.cfg.Task,
		matchTasks:  s.cfg.MatchTasks,
		status:      s.cfg.Status,
		table:       s.table,
		dispatcher:  s.state.Dispatcher(),
		sweeper:     s.sweeper,
		resyncs:     s.resyncs,
		deletes:     s.deletes,
	}
}

// NewRetrieve opens a new query builder for retrieving Arcs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}
