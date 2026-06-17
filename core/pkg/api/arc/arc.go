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
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/gorp"
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

// Dispatch relays the action sequence to the other clients editing the arc, broadcasting
// it on the arc collaborative-edit signals channel. The caller must hold update access
// to the arc. The server does not interpret or persist the actions.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{arc.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type (
	RetrieveRequest struct {
		SearchTerm    string    `json:"search_term" msgpack:"search_term"`
		Keys          []arc.Key `json:"keys" msgpack:"keys"`
		Names         []string  `json:"names" msgpack:"names"`
		Limit         int       `json:"limit" msgpack:"limit"`
		Offset        int       `json:"offset" msgpack:"offset"`
		IncludeStatus bool      `json:"include_status" msgpack:"include_status"`
		Compile       bool      `json:"compile" msgpack:"compile"`
		IncludeDoc    bool      `json:"include_doc" msgpack:"include_doc"`
	}
	RetrieveResponse struct {
		Arcs []Arc         `json:"arcs" msgpack:"arcs"`
		Docs []DocSnapshot `json:"docs" msgpack:"docs"`
	}
	// DocSnapshot is the set of operations that reconstruct an arc's live collaborative
	// document. Retrieve returns one per arc when IncludeDoc is set so a joining client
	// bootstraps into the same id space as the other editors instead of re-seeding from
	// the materialized text and diverging.
	DocSnapshot struct {
		Key     arc.Key       `json:"key" msgpack:"key"`
		Inserts []crdt.Insert `json:"inserts" msgpack:"inserts"`
		Deletes []crdt.Delete `json:"deletes" msgpack:"deletes"`
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
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}

	res := RetrieveResponse{Arcs: arcs}

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

	if req.IncludeDoc {
		res.Docs = make([]DocSnapshot, 0, len(arcs))
		for i := range arcs {
			inserts, deletes, err := s.internal.Snapshot(ctx, arcs[i].Key)
			if err != nil {
				return RetrieveResponse{}, err
			}
			res.Docs = append(res.Docs, DocSnapshot{
				Key:     arcs[i].Key,
				Inserts: inserts,
				Deletes: deletes,
			})
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
	parsed, diag := arctext.Parse(arc.Text)
	if diag != nil && !diag.Ok() {
		return CompileError{Diagnostics: diag.Error()}
	}
	ir, diag := arctext.Analyze(ctx, parsed, s.internal.NewRoot(nil))
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
