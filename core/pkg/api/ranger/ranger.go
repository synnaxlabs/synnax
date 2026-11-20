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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	// Key is a unique identifier for a range.
	Key = ranger.Key
	// Range is a user-defined region of time.
	Range = ranger.Range
)

// ServiceConfig is the configuration for the range API service.
type ServiceConfig struct {
	// Ranger stores and retrieves ranges.
	//
	// [REQUIRED]
	Ranger *ranger.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the range API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger")
	v.NotNil("ranger", c.Ranger)
	return v.Error()
}

type Service struct {
	internal *ranger.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Ranger,
	}, nil
}

type (
	CreateRequest struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

// Create creates the ranges. It returns a validation error when a range with one of
// their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Ranges); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Ranges: req.Ranges}, nil
}

// DispatchRequest carries an action sequence to apply to a single range. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[ranger.Key, ranger.Action]

// Dispatch applies the action sequence to the target range atomically. Subscribers to
// the range action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the ranges that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, ranger.ResourceID)
}

// Retrieve returns the ranges that req selects and that the subject can retrieve. A
// child range passes only when the subject can also retrieve its parent, and a named
// child with a denied parent fails the request with an access error. HasChildren counts
// only the children that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Ranges))
	if err != nil {
		return RetrieveResponse{}, err
	}
	named := req.Named()
	if err = q.Where(ranger.Match(func(
		_ gorp.Context,
		_ ranger.Retrieve,
		r *ranger.Range,
	) (bool, error) {
		if !snap.Allowed(r.ResourceID()) {
			return false, nil
		}
		if r.Parent == nil {
			return true, nil
		}
		parent := ranger.ResourceID(*r.Parent)
		if snap.Allowed(parent) {
			return true, nil
		}
		if slices.Contains(named, r.ResourceID()) {
			return false, snap.Enforce(parent)
		}
		return false, nil
	})).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.internal.Derive(ctx, snap, res.Ranges, func(k ranger.Key) bool {
		return snap.Allowed(ranger.ResourceID(k))
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []ranger.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the ranges with the given keys, the ranges nested in them, and the
// records they own. A key that matches no range is ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
