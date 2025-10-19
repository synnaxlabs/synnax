// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

type Arc struct {
	arc.Arc
	Status status.Status `json:"status" msgpack:"status"`
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
		Action:  access.Create,
		Objects: arc.OntologyIDsFromArcs(translateArcsToService(req.Arcs)),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, arc_ := range req.Arcs {
			if err = w.Create(ctx, &arc_.Arc); err != nil {
				return err
			}
			req.Arcs[i] = arc_
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
		Action:  access.Delete,
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
		IncludeStatus *bool       `json:"include_status" msgpack:"include_status"`
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

	if req.IncludeStatus != nil && *req.IncludeStatus {
		statuses := make([]status.Status, 0, len(res.Arcs))
		uuidStrings := lo.Map(req.Keys, func(item uuid.UUID, _ int) string {
			return item.String()
		})
		if err = s.status.NewRetrieve().WhereKeys(uuidStrings...).Entries(&statuses).Exec(ctx, nil); err != nil {
			return ArcRetrieveResponse{}, err
		}
		for i, stat := range statuses {
			res.Arcs[i].Status = stat
		}
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
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
		Instrumentation: s.Child("arc_lsp"),
	})
	if err != nil {
		return err
	}
	return arctransport.ServeFreighter(ctx, lspServer, stream)
}
