// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/agent"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *agent.Service
	alamos.Instrumentation
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:              cfg.Distribution.DB,
		access:          cfg.Service.RBAC,
		Instrumentation: cfg.Instrumentation,
		internal:        cfg.Service.Agent,
	}, nil
}

type (
	CreateRequest struct {
		Agents []agent.Agent `json:"agents" msgpack:"agents"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: agent.OntologyIDsFromAgents(req.Agents),
	}); err != nil {
		return res, err
	}
	// Phase 1: Save agents in StateGenerating within a short-lived transaction.
	if err = s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, a := range req.Agents {
			if err = w.CreatePending(ctx, &a); err != nil {
				return err
			}
			req.Agents[i] = a
		}
		return nil
	}); err != nil {
		return res, err
	}
	// Phase 2: Generate Arc code and deploy outside the transaction.
	for i := range req.Agents {
		if len(req.Agents[i].Messages) > 0 {
			if err = s.internal.GenerateAndDeploy(ctx, &req.Agents[i]); err != nil {
				res.Agents = req.Agents
				return res, err
			}
		}
	}
	res.Agents = req.Agents
	return res, nil
}

type (
	SendRequest struct {
		Key     uuid.UUID `json:"key" msgpack:"key"`
		Content string    `json:"content" msgpack:"content"`
	}
	SendResponse struct {
		Agent agent.Agent `json:"agent" msgpack:"agent"`
	}
)

func (s *Service) Send(ctx context.Context, req SendRequest) (res SendResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: agent.OntologyIDs([]uuid.UUID{req.Key}),
	}); err != nil {
		return res, err
	}
	// Phase 1: Append message and save in StateGenerating.
	var a *agent.Agent
	if err = s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		a, err = w.PrepareSend(ctx, req.Key, req.Content)
		return err
	}); err != nil {
		return res, err
	}
	// Phase 2: Generate Arc code and deploy outside the transaction.
	if err = s.internal.GenerateAndDeploy(ctx, a); err != nil {
		res.Agent = *a
		return res, err
	}
	s.L.Info("send response",
		zap.Stringer("key", a.Key),
		zap.Stringer("arc_key", a.ArcKey),
		zap.String("state", string(a.State)),
		zap.Int("messages", len(a.Messages)),
	)
	res.Agent = *a
	return res, nil
}

type (
	RetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Agents []agent.Agent `json:"agents" msgpack:"agents"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	q := s.internal.NewRetrieve().Entries(&res.Agents)
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if err = q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: agent.OntologyIDsFromAgents(res.Agents),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: agent.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
