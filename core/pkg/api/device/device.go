// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	db     *gorp.DB
	access *rbac.Service
	device *device.Service
	status *status.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:     cfg.Distribution.DB,
		access: cfg.Service.RBAC,
		device: cfg.Service.Device,
		status: cfg.Service.Status,
	}
}

type (
	CreateRequest struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
	CreateResponse struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
)

func (svc *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: device.OntologyIDsFromDevices(req.Devices),
	}); err != nil {
		return res, err
	}
	return res, svc.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.device.NewWriter(tx)
		for _, d := range req.Devices {
			if err := w.Create(ctx, d); err != nil {
				return err
			}
		}
		res.Devices = req.Devices
		return nil
	})
}

type RetrieveRequest struct {
	SearchTerm     string     `json:"search_term" msgpack:"search_term"`
	Keys           []string   `json:"keys" msgpack:"keys"`
	Names          []string   `json:"names" msgpack:"names"`
	Makes          []string   `json:"makes" msgpack:"makes"`
	Models         []string   `json:"models" msgpack:"models"`
	Locations      []string   `json:"locations" msgpack:"locations"`
	Racks          []rack.Key `json:"racks" msgpack:"racks"`
	Limit          int        `json:"limit" msgpack:"limit"`
	Offset         int        `json:"offset" msgpack:"offset"`
	IgnoreNotFound bool       `json:"ignore_not_found" msgpack:"ignore_not_found"`
	IncludeStatus  bool       `json:"include_status" msgpack:"include_status"`
}

type RetrieveResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, _ error) {
	var (
		hasSearch    = len(req.SearchTerm) > 0
		hasKeys      = len(req.Keys) > 0
		hasNames     = len(req.Names) > 0
		hasMakes     = len(req.Makes) > 0
		hasLimit     = req.Limit > 0
		hasOffset    = req.Offset > 0
		hasLocations = len(req.Locations) > 0
		hasModels    = len(req.Models) > 0
		hasRacks     = len(req.Racks) > 0
	)
	q := svc.device.NewRetrieve()
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if hasMakes {
		q = q.WhereMakes(req.Makes...)
	}
	if hasLocations {
		q = q.WhereLocations(req.Locations...)
	}
	if hasModels {
		q = q.WhereModels(req.Models...)
	}
	if hasRacks {
		q = q.WhereRacks(req.Racks...)
	}
	retErr := q.Entries(&res.Devices).Exec(ctx, nil)

	if req.IncludeStatus {
		statuses := make([]device.Status, 0, len(res.Devices))
		if err := status.NewRetrieve[device.StatusDetails](svc.status).
			WhereKeys(ontology.IDsToKeys(device.OntologyIDsFromDevices(res.Devices))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			res.Devices[i].Status = &stat
		}
	}

	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: device.OntologyIDsFromDevices(res.Devices),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	if retErr != nil && req.IgnoreNotFound {
		retErr = errors.Skip(retErr, query.ErrNotFound)
	}
	return res, retErr
}

type DeleteRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (svc *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: device.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.device.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}
