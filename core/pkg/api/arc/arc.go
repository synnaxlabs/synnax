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

	"github.com/synnaxlabs/alamos"
	arclang "github.com/synnaxlabs/arc"
	arctransport "github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	Arc = arc.Arc
	Key = arc.Key
)

// ServiceConfig is the configuration for the Arc API service.
type ServiceConfig struct {
	// DB holds the channels and modules that compilation reads. Compilation reads it
	// with no snapshot, so it delays no write.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Arc stores and runs Arc automations.
	//
	// [REQUIRED]
	Arc *arc.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC reads the access of the subject of each LSP stream.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the Arc API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.arc")
	v.NotNil("db", c.DB)
	v.NotNil("arc", c.Arc)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	db       *gorp.DB
	internal *arc.Service
	status   *status.Service
	rbac     *rbac.Service
	alamos.Instrumentation
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
		internal:        cfg.Arc,
		status:          cfg.Status,
		rbac:            cfg.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Arcs); err != nil {
		return CreateResponse{}, err
	}
	for i := range req.Arcs {
		req.Arcs[i] = materialize(req.Arcs[i])
	}
	return CreateResponse{Arcs: req.Arcs}, nil
}

type DeleteRequest struct {
	Keys []arc.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// DispatchRequest carries a sequence of collaborative-edit actions to relay to the
// other clients editing a single arc. DispatchKey is the originating client's batch
// identifier, echoed verbatim on the broadcast so the sender can recognize its own
// edits.
type DispatchRequest = actions.DispatchRequest[arc.Key, arc.Action]

// Dispatch applies the action sequence to the arc in tx. Subscribers to the arc action
// signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

type (
	// UpdateTaskRequest changes the arc's task. A nil field stays as it is.
	UpdateTaskRequest struct {
		// Key is the key of the arc.
		Key arc.Key `json:"key" msgpack:"key"`
		// Rack binds the arc to the rack. A zero rack unbinds it.
		Rack *rack.Key `json:"rack,omitempty" msgpack:"rack,omitempty"`
		// Performance sets how closely the task's runtime holds timer deadlines.
		Performance *arctask.Performance `json:"performance,omitempty" msgpack:"performance,omitempty"`
	}
	// UpdateTaskResponse carries the arc's task, or a null Task after an unbind.
	UpdateTaskResponse struct {
		Task *task.Task `json:"task" msgpack:"task"`
	}
)

// UpdateTask creates, moves, or changes the arc's task. A zero rack unbinds the arc,
// deleting its task; unbinding a running arc is rejected.
func (s *Service) UpdateTask(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateTaskRequest,
) (UpdateTaskResponse, error) {
	tsk, err := s.internal.NewWriter(tx).UpdateTask(ctx, req.Key, arc.TaskUpdate{
		Rack:        req.Rack,
		Performance: req.Performance,
	})
	if err != nil {
		return UpdateTaskResponse{}, err
	}
	return UpdateTaskResponse{Task: tsk}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the Arcs that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, arc.ResourceID)
}

// Retrieve returns the Arcs that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Arcs))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, arc.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	// Raw is derived from the replicated document and not stored, so materialize it
	// before compilation and for clients that read the source text directly rather than
	// reconstructing the document.
	for i := range res.Arcs {
		res.Arcs[i] = materialize(res.Arcs[i])
	}
	return res, nil
}

// Compiled returns a handler that runs retrieve and then, when the request includes
// programs, compiles each automation that retrieve returns. Compilation runs after
// retrieve returns, so it holds no read lock and delays no write.
func (s *Service) Compiled(
	retrieve freighter.UnaryHandler[RetrieveRequest, RetrieveResponse],
) freighter.UnaryHandler[RetrieveRequest, RetrieveResponse] {
	return func(ctx context.Context, req RetrieveRequest) (RetrieveResponse, error) {
		res, err := retrieve(ctx, req)
		if err != nil || !req.IncludeProgram {
			return res, err
		}
		for i := range res.Arcs {
			if err = s.compile(ctx, &res.Arcs[i]); err != nil {
				return RetrieveResponse{}, err
			}
		}
		return res, nil
	}
}

// LSPMessage represents a single JSON-RPC message for the LSP
type LSPMessage = arctransport.JSONRPCMessage

// LSP serves the Arc language server over stream. The server treats the channels and
// modules that the subject of the stream cannot retrieve as unknown names.
func (s *Service) LSP(
	ctx context.Context,
	stream freighter.ServerStream[LSPMessage, LSPMessage],
) error {
	subject := auth.GetSubject(ctx)
	lsp, err := s.internal.NewLSP(arc.LSPConfig{
		Allowed: func(ctx context.Context) (allowed func(resource.ID) bool, err error) {
			err = s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				allowed, err = s.rbac.NewEnforcer(snap).
					Allowed(ctx, subject, access.ActionRetrieve)
				return err
			})
			return allowed, err
		},
	})
	if err != nil {
		return err
	}
	return arctransport.ServeFreighter(ctx, arctransport.Config{
		Server: lsp,
		Stream: stream,
	})
}

// materialize returns a with the raw text of a text-mode Arc derived from its document.
func materialize(a Arc) Arc {
	b := a.Base()
	src, ok := b.Source.Variant.(arc.TextSource)
	if !ok {
		return a
	}
	src.Text = src.Text.Materialize()
	b.Source.Variant = src
	a.SetBase(b)
	return a
}

// compile compiles the source of an automation into its program and leaves a module as
// is. Returns an error if parsing, analysis, or compilation fails.
func (s *Service) compile(ctx context.Context, a *Arc) error {
	automation, ok := a.Variant.(arc.AutomationArc)
	if !ok {
		return nil
	}
	prog, err := channel.CompileConsistent(ctx, s.db, func(
		snap gorp.Snapshot,
		lookups *channel.Lookups,
	) (arclang.Program, error) {
		root := s.internal.NewRoot(snap, lookups)
		switch src := automation.Source.Variant.(type) {
		case arc.TextSource:
			return s.compileText(ctx, src.Text, root)
		case arc.GraphSource:
			return arclang.CompileGraph(
				ctx,
				src.Graph,
				root,
				arclang.WithAllowDashedNames(s.internal.AllowDashedNames()),
			)
		}
		return arclang.Program{}, nil
	})
	if err != nil {
		return err
	}
	automation.Program = &prog
	a.Variant = automation
	return nil
}

// compileText compiles t against root, reporting parse and analysis failures as a
// CompileError.
func (s *Service) compileText(
	ctx context.Context,
	t arctext.Text,
	root *symbol.Symbol,
) (arclang.Program, error) {
	cfg := parser.Config{AllowDashedNames: s.internal.AllowDashedNames()}
	parsed, diag := arctext.Parse(t, cfg)
	if diag != nil && !diag.Ok() {
		return arclang.Program{}, CompileError{Diagnostics: diag.Error()}
	}
	ir, diag := arctext.Analyze(ctx, parsed, root, cfg)
	if diag != nil && !diag.Ok() {
		return arclang.Program{}, CompileError{Diagnostics: diag.Error()}
	}
	return arctext.Compile(ctx, ir)
}
