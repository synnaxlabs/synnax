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
	"go/types"

	"github.com/synnaxlabs/alamos"
	arctransport "github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/arc/parser"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type (
	Arc = arc.Arc
	Key = arc.Key
)

type Service struct {
	access   *rbac.Service
	internal *arc.Service
	status   *status.Service
	alamos.Instrumentation
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:          cfg.Service.RBAC,
		Instrumentation: cfg.Instrumentation,
		internal:        cfg.Service.Arc,
		status:          cfg.Service.Status,
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
	return CreateResponse(req), nil
}

type DeleteRequest struct {
	Keys []arc.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: arc.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// DispatchRequest carries a sequence of collaborative-edit actions to relay to the
// other clients editing a single arc. DispatchKey is the originating client's batch
// identifier, echoed verbatim on the broadcast so the sender can recognize its own
// edits.
type DispatchRequest = actions.DispatchRequest[arc.Key, arc.Action]

// DispatchResponse carries the arc's semantic hash after the dispatched actions were
// applied, letting the editing client refresh its staleness signal without a refetch.
type DispatchResponse struct {
	Hash string `json:"hash" msgpack:"hash"`
}

// Dispatch relays the action sequence to the other clients editing the arc, broadcasting
// it on the arc collaborative-edit signals channel. The caller must hold update access
// to the arc.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{arc.OntologyID(req.Key)},
	}); err != nil {
		return DispatchResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions); err != nil {
		return DispatchResponse{}, err
	}
	var updated Arc
	if err := s.internal.NewRetrieve().
		Where(arc.MatchKeys(req.Key)).
		Entry(&updated).
		Exec(ctx, tx); err != nil {
		return DispatchResponse{}, err
	}
	hash, err := arc.Hash(updated)
	if err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Hash: hash}, nil
}

type (
	// DeployRequest binds the arc to a rack. A zero Rack undeploys the arc.
	DeployRequest struct {
		Key  arc.Key  `json:"key" msgpack:"key"`
		Rack rack.Key `json:"rack" msgpack:"rack"`
	}
	// DeployResponse carries the deployed task, or a nil Task after an undeploy.
	DeployResponse struct {
		Task *task.Task `json:"task,omitempty" msgpack:"task,omitempty"`
	}
)

// Deploy creates or moves the arc's task so it runs on the requested rack, stamping
// the arc's current semantic hash into the task config. A zero rack undeploys the arc,
// deleting its task; undeploying a running arc is rejected.
func (s *Service) Deploy(
	ctx context.Context,
	tx gorp.Tx,
	req DeployRequest,
) (DeployResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{arc.OntologyID(req.Key)},
	}); err != nil {
		return DeployResponse{}, err
	}
	taskAction := access.ActionCreate
	if req.Rack == 0 {
		taskAction = access.ActionDelete
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  taskAction,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeTask}},
	}); err != nil {
		return DeployResponse{}, err
	}
	tsk, err := s.internal.NewWriter(tx).Deploy(ctx, req.Key, req.Rack)
	if err != nil {
		return DeployResponse{}, err
	}
	return DeployResponse{Task: tsk}, nil
}

type (
	RetrieveRequest struct {
		SearchTerm          string    `json:"search_term" msgpack:"search_term"`
		Keys                []arc.Key `json:"keys" msgpack:"keys"`
		Names               []string  `json:"names" msgpack:"names"`
		Limit               int       `json:"limit" msgpack:"limit"`
		Offset              int       `json:"offset" msgpack:"offset"`
		IncludeStatus       bool      `json:"include_status" msgpack:"include_status"`
		Compile             bool      `json:"compile" msgpack:"compile"`
		IgnoreNotFoundError bool      `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
	}
	RetrieveResponse struct {
		Arcs []Arc `json:"arcs,omitzero" msgpack:"arcs,omitzero"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var arcs []arc.Arc
	var (
		q         = s.internal.NewRetrieve().Entries(&arcs)
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasSearch = req.SearchTerm != ""
	)
	if hasKeys {
		q = q.Where(arc.MatchKeys(req.Keys...))
	}
	if hasNames {
		q = q.Where(arc.MatchNames(req.Names...))
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	err := q.Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}

	res := RetrieveResponse{Arcs: arcs}

	// Raw is derived from the replicated document and not stored, so materialize it before
	// compilation and for clients that read the source text directly rather than
	// reconstructing the document.
	for i := range res.Arcs {
		res.Arcs[i].Text = res.Arcs[i].Text.Materialize()
		hash, err := arc.Hash(res.Arcs[i])
		if err != nil {
			return RetrieveResponse{}, err
		}
		res.Arcs[i].Hash = &hash
	}

	// Compile Arcs to modules if requested
	if req.Compile {
		for i := range res.Arcs {
			if err := s.compile(ctx, &res.Arcs[i]); err != nil {
				return RetrieveResponse{}, err
			}
		}
	} else {
		// Reset the program to zero value just in case it was there
		for i := range res.Arcs {
			res.Arcs[i].Program = nil
		}
	}

	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: arc.OntologyIDsFromArcs(arcs),
	}); err != nil {
		return RetrieveResponse{}, err
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
