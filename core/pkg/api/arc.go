// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	arclsp "github.com/synnaxlabs/arc/lsp"
	arctransport "github.com/synnaxlabs/arc/lsp/transport"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

type Arc struct {
	arc.Arc
	Status *status.Status[arc.StatusDetails] `json:"status" msgpack:"status"`
}

type ArcService struct {
	dbProvider
	accessProvider
	alamos.Instrumentation
	internal *arc.Service
	status   *status.Service
}

func NewArcService(p Provider) *ArcService {
	return &ArcService{
		dbProvider:      p.db,
		accessProvider:  p.access,
		Instrumentation: p.Instrumentation,
		internal:        p.Service.Arc,
		status:          p.Service.Status,
	}
}

type (
	ArcCreateRequest struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
	ArcCreateResponse = ArcCreateRequest
)

func (s *ArcService) Create(ctx context.Context, req ArcCreateRequest) (res ArcCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: arc.OntologyIDsFromArcs(translateArcsToService(req.Arcs)),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, a := range req.Arcs {
			if err = w.Create(ctx, &a.Arc); err != nil {
				return err
			}
			req.Arcs[i] = a
		}
		res.Arcs = req.Arcs
		return nil
	})
}

type ArcDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *ArcService) Delete(ctx context.Context, req ArcDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: arc.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	ArcRetrieveRequest struct {
		Keys          []uuid.UUID `json:"keys" msgpack:"keys"`
		Names         []string    `json:"names" msgpack:"names"`
		SearchTerm    string      `json:"search_term" msgpack:"search_term"`
		Limit         int         `json:"limit" msgpack:"limit"`
		Offset        int         `json:"offset" msgpack:"offset"`
		IncludeStatus bool        `json:"include_status" msgpack:"include_status"`
		Compile       bool        `json:"compile" msgpack:"compile"`
	}
	ArcRetrieveResponse struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
)

func (s *ArcService) Retrieve(ctx context.Context, req ArcRetrieveRequest) (res ArcRetrieveResponse, err error) {
	var (
		svcArcs   []arc.Arc
		q         = s.internal.NewRetrieve().Entries(&svcArcs)
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasSearch = req.SearchTerm != ""
	)

	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
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

	if err = q.Exec(ctx, nil); err != nil {
		return ArcRetrieveResponse{}, err
	}

	res.Arcs = translateArcsFromService(svcArcs)

	// Compile Arcs to modules if requested
	if req.Compile {
		for i := range res.Arcs {
			if err = s.compileArc(ctx, &res.Arcs[i]); err != nil {
				return ArcRetrieveResponse{}, err
			}
		}
	}

	if req.IncludeStatus {
		statuses := make([]status.Status[arc.StatusDetails], 0, len(res.Arcs))
		uuidStrings := lo.Map(res.Arcs, func(a Arc, _ int) string {
			return a.Key.String()
		})
		if err = status.NewRetrieve[arc.StatusDetails](s.status).
			WhereKeys(uuidStrings...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return ArcRetrieveResponse{}, err
		}
		for i, stat := range statuses {
			res.Arcs[i].Status = &stat
		}
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: arc.OntologyIDsFromArcs(svcArcs),
	}); err != nil {
		return ArcRetrieveResponse{}, err
	}
	return res, nil
}

func translateArcsToService(arcs []Arc) []arc.Arc {
	return lo.Map(arcs, func(a Arc, _ int) arc.Arc { return a.Arc })
}

func translateArcsFromService(arcs []arc.Arc) []Arc {
	return lo.Map(arcs, func(a arc.Arc, _ int) Arc { return Arc{Arc: a} })
}

// ArcLSPMessage represents a single JSON-RPC message for the LSP
type ArcLSPMessage = arctransport.JSONRPCMessage

// LSP handles LSP protocol messages over a Freighter stream
func (s *ArcService) LSP(ctx context.Context, stream freighter.ServerStream[ArcLSPMessage, ArcLSPMessage]) error {
	// Create a new LSP server instance for this connection with a no-op logger
	// to avoid nil pointer panics
	lspServer, err := arclsp.New(arclsp.Config{
		Instrumentation: s.Child("arc").Child("lsp"),
		GlobalResolver:  s.internal.SymbolResolver(),
	})
	if err != nil {
		return err
	}
	return arctransport.ServeFreighter(ctx, lspServer, stream)
}

// compileArc compiles the Arc text to a module containing IR and WASM bytecode.
// Returns an error if parsing, analysis, or compilation fails.
func (s *ArcService) compileArc(ctx context.Context, arc *Arc) error {
	// Step 1: Parse the Arc text
	parsed, diag := arctext.Parse(arc.Text)
	if diag != nil && !diag.Ok() {
		return errors.Newf("parse failed for arc %s: %w", arc.Key, diag)
	}

	// Step 2: Analyze the parsed text to produce IR
	ir, diag := arctext.Analyze(ctx, parsed, s.internal.SymbolResolver())
	if diag != nil && !diag.Ok() {
		return errors.Newf("analysis failed for arc %s: %w", arc.Key, diag)
	}

	// Step 3: Compile IR to WebAssembly module
	mod, err := arctext.Compile(ctx, ir)
	if err != nil {
		return errors.Newf("compilation failed for arc %s: %w", arc.Key, err)
	}

	// Step 4: Attach compiled module to Arc
	arc.Module = mod
	return nil
}
