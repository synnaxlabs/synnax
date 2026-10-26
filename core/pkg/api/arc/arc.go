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
	arctransport "github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/arc/parser"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
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
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Arc stores and runs Arc automations.
	//
	// [REQUIRED]
	Arc *arc.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the Arc API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.arc")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("arc", c.Arc)
	v.NotNil("status", c.Status)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *arc.Service
	status   *status.Service
	alamos.Instrumentation
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:          cfg.RBAC,
		Instrumentation: cfg.Instrumentation,
		internal:        cfg.Arc,
		status:          cfg.Status,
	}, nil
}

type (
	CreateRequest struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeArc}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Arcs); err != nil {
		return CreateResponse{}, err
	}
	for i := range req.Arcs {
		req.Arcs[i].Text = req.Arcs[i].Text.Materialize()
	}
	return CreateResponse(req), nil
}

type DeleteRequest struct {
	Keys []arc.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: arc.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// DispatchRequest carries a sequence of collaborative-edit actions to relay to the
// other clients editing a single arc. DispatchKey is the originating client's batch
// identifier, echoed verbatim on the broadcast so the sender can recognize its own
// edits.
type DispatchRequest = actions.DispatchRequest[arc.Key, arc.Action]

// Dispatch relays the action sequence to the other clients editing the arc,
// broadcasting it on the arc collaborative-edit signals channel. The caller must hold
// update access to the arc.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{arc.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{arc.OntologyID(req.Key)},
	}); err != nil {
		return UpdateTaskResponse{}, err
	}
	taskAction := access.ActionUpdate
	if req.Rack != nil {
		taskAction = access.ActionCreate
		if *req.Rack == 0 {
			taskAction = access.ActionDelete
		}
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  taskAction,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeTask}},
	}); err != nil {
		return UpdateTaskResponse{}, err
	}
	tsk, err := s.internal.NewWriter(tx).UpdateTask(ctx, req.Key, arc.TaskUpdate{
		Rack:        req.Rack,
		Performance: req.Performance,
	})
	if err != nil {
		return UpdateTaskResponse{}, err
	}
	return UpdateTaskResponse{Task: tsk}, nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Arcs))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: arc.OntologyIDsFromArcs(res.Arcs),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	// Raw is derived from the replicated document and not stored, so materialize it
	// before compilation and for clients that read the source text directly rather than
	// reconstructing the document.
	for i := range res.Arcs {
		res.Arcs[i].Text = res.Arcs[i].Text.Materialize()
		if req.IncludeProgram {
			if err := s.compile(ctx, &res.Arcs[i]); err != nil {
				return RetrieveResponse{}, err
			}
		}
	}
	return res, nil
}

// LSPMessage represents a single JSON-RPC message for the LSP
type LSPMessage = arctransport.JSONRPCMessage

// LSP handles LSP protocol messages over a Freighter stream
func (s *Service) LSP(
	ctx context.Context,
	stream freighter.ServerStream[LSPMessage, LSPMessage],
) error {
	lsp, err := s.internal.NewLSP()
	if err != nil {
		return err
	}
	return arctransport.ServeFreighter(ctx, arctransport.Config{
		Server: lsp,
		Stream: stream,
	})
}

// compile compiles the Arc text to a module containing IR and WASM bytecode. Returns an
// error if parsing, analysis, or compilation fails.
func (s *Service) compile(ctx context.Context, arc *Arc) error {
	cfg := parser.Config{AllowDashedNames: s.internal.AllowDashedNames()}
	parsed, diag := arctext.Parse(arc.Text, cfg)
	if diag != nil && !diag.Ok() {
		return CompileError{Diagnostics: diag.Error()}
	}
	ir, diag := arctext.Analyze(ctx, parsed, s.internal.NewRoot(nil), cfg)
	if diag != nil && !diag.Ok() {
		return CompileError{Diagnostics: diag.Error()}
	}
	mod, err := arctext.Compile(ctx, ir)
	if err != nil {
		return err
	}
	arc.Program = &mod
	return nil
}
