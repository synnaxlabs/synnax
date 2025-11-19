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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type RackService struct {
	dbProvider
	accessProvider
	rack   *rack.Service
	device *device.Service
	task   *task.Service
	status statusProvider
}

func NewRackService(p Provider) *RackService {
	return &RackService{
		dbProvider:     p.db,
		rack:           p.Service.Rack,
		device:         p.Service.Device,
		task:           p.Service.Task,
		status:         p.status,
		accessProvider: p.access,
	}
}

type (
	Rack = rack.Rack
)

type (
	RackCreateRequest struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
	RackCreateResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (svc *RackService) CreateRack(ctx context.Context, req RackCreateRequest) (res RackCreateResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: rack.OntologyIDsFromRacks(req.Racks),
	}); err != nil {
		return res, err
	}
	if err := svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.rack.NewWriter(tx)
		for i, r := range req.Racks {
			if err := w.Create(ctx, &r); err != nil {
				return err
			}
			req.Racks[i] = r
		}
		res.Racks = req.Racks
		return nil
	}); err != nil {
		return RackCreateResponse{}, err
	}
	return res, nil
}

type (
	RackRetrieveRequest struct {
		Keys          []rack.Key `json:"keys" msgpack:"keys"`
		Names         []string   `json:"names" msgpack:"names"`
		SearchTerm    string     `json:"search_term" msgpack:"search_term"`
		Embedded      *bool      `json:"embedded" msgpack:"embedded"`
		HostIsNode    *bool      `json:"host_is_node" msgpack:"host_is_node"`
		Limit         int        `json:"limit" msgpack:"limit"`
		Offset        int        `json:"offset" msgpack:"offset"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
	}
	RackRetrieveResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (svc *RackService) RetrieveRack(ctx context.Context, req RackRetrieveRequest) (res RackRetrieveResponse, _ error) {
	var (
		hasSearch = len(req.SearchTerm) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	resRacks := make([]rack.Rack, 0, len(req.Keys)+len(req.Names))
	q := svc.rack.NewRetrieve()
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if req.Embedded != nil {
		q = q.WhereEmbedded(*req.Embedded)
	}
	if req.HostIsNode != nil {
		q = q.WhereNodeIsHost(*req.HostIsNode)
	}
	if err := q.Entries(&resRacks).Exec(ctx, nil); err != nil {
		return res, err
	}

	if req.IncludeStatus {
		keys := make([]rack.Key, len(resRacks))
		for i := range resRacks {
			keys[i] = resRacks[i].Key
		}
		statuses := make([]rack.Status, 0, len(resRacks))
		if err := status.NewRetrieve[rack.StatusDetails](svc.status.status).
			WhereKeys(ontology.IDsToString(rack.OntologyIDsFromRacks(resRacks))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			resRacks[i].Status = &stat
		}
	}

	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: rack.OntologyIDsFromRacks(resRacks),
	}); err != nil {
		return res, err
	}
	res.Racks = resRacks
	return res, nil
}

type RackDeleteRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

func embeddedGuard(_ gorp.Context, r Rack) error {
	if !r.Embedded {
		return nil
	}
	return errors.Wrapf(validate.Error, "cannot delete embedded rack")
}

func (svc *RackService) DeleteRack(
	ctx context.Context,
	req RackDeleteRequest,
) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: rack.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		exists, err := svc.device.NewRetrieve().WhereRacks(req.Keys...).Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(validate.Error, "cannot delete rack when devices are still attached")
		}
		exists, err = svc.task.NewRetrieve().WhereInternal(false, gorp.Required()).WhereRacks(req.Keys...).Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(validate.Error, "cannot delete rack when tasks are still attached")
		}
		w := svc.rack.NewWriter(tx)
		for _, k := range req.Keys {
			if err = w.DeleteGuard(ctx, k, embeddedGuard); err != nil {
				return err
			}
		}
		return nil
	})
}
