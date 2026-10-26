// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	// Key is a unique identifier for a range.
	Key = ranger.Key
	// Range is a user-defined region of time.
	Range = ranger.Range
)

func rangeAccessOntologyIDs(ranges []Range) []ontology.ID {
	ids := make([]ontology.ID, 0, len(ranges))
	for _, r := range ranges {
		ids = append(ids, r.OntologyID())
		ids = append(ids, label.OntologyIDsFromLabels(r.Labels)...)
		if r.Parent == nil {
			continue
		}
		ids = append(ids, r.Parent.OntologyID())
	}
	return ids
}

// ServiceConfig is the configuration for the range API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Ranger stores and retrieves ranges.
	//
	// [REQUIRED]
	Ranger *ranger.Service
	// Label stores and retrieves labels.
	//
	// [REQUIRED]
	Label *label.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the range API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	c.Label = override.Nil(c.Label, other.Label)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("ranger", c.Ranger)
	v.NotNil("label", c.Label)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *ranger.Service
	label    *label.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.Ranger,
		label:    cfg.Label,
	}, nil
}

type (
	CreateRequest struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
	CreateResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: rangeAccessOntologyIDs(req.Ranges),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Ranges); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Ranges))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeLabels {
		for i, r := range res.Ranges {
			labels, err := s.label.RetrieveFor(ctx, r.OntologyID(), nil)
			if err != nil {
				return RetrieveResponse{}, err
			}
			res.Ranges[i].Labels = labels
		}
	}
	if req.IncludeParent {
		for i, r := range res.Ranges {
			parentKey, err := s.internal.RetrieveParentKey(ctx, r.Key, nil)
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			if err != nil {
				return RetrieveResponse{}, err
			}
			res.Ranges[i].Parent = &ranger.Range{}
			if err = s.
				internal.
				NewRetrieve().
				Entry(res.Ranges[i].Parent).
				Where(ranger.MatchKeys(parentKey)).
				Exec(ctx, nil); err != nil {
				return RetrieveResponse{}, err
			}
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rangeAccessOntologyIDs(res.Ranges),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type RenameRequest struct {
	Name string     `json:"name" msgpack:"name"`
	Key  ranger.Key `json:"key"  msgpack:"key"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type SetEndRequest struct {
	Key ranger.Key      `json:"key" msgpack:"key"`
	End telem.TimeStamp `json:"end" msgpack:"end"`
}

func (s *Service) SetEnd(
	ctx context.Context,
	tx gorp.Tx,
	req SetEndRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetEnd(ctx, req.Key, req.End)
}

type DeleteRequest struct {
	Keys []ranger.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ranger.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
