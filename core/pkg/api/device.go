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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type DeviceService struct {
	dbProvider
	accessProvider
	device *device.Service
	status statusProvider
}

func NewDeviceService(p Provider) *DeviceService {
	return &DeviceService{
		dbProvider:     p.db,
		device:         p.Service.Device,
		status:         p.status,
		accessProvider: p.access,
	}
}

type (
	Device = device.Device
)

type DeviceCreateRequest struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

type DeviceCreateResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *DeviceService) Create(ctx context.Context, req DeviceCreateRequest) (res DeviceCreateResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: device.OntologyIDsFromDevices(req.Devices),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
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

type DeviceRetrieveRequest struct {
	Keys           []string   `json:"keys" msgpack:"keys"`
	Names          []string   `json:"names" msgpack:"names"`
	Makes          []string   `json:"makes" msgpack:"makes"`
	Models         []string   `json:"models" msgpack:"models"`
	Locations      []string   `json:"locations" msgpack:"locations"`
	Racks          []rack.Key `json:"racks" msgpack:"racks"`
	SearchTerm     string     `json:"search_term" msgpack:"search_term"`
	Limit          int        `json:"limit" msgpack:"limit"`
	Offset         int        `json:"offset" msgpack:"offset"`
	IgnoreNotFound bool       `json:"ignore_not_found" msgpack:"ignore_not_found"`
	IncludeStatus  bool       `json:"include_status" msgpack:"include_status"`
}

type DeviceRetrieveResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *DeviceService) Retrieve(ctx context.Context, req DeviceRetrieveRequest) (res DeviceRetrieveResponse, _ error) {
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
		if err := status.NewRetrieve[device.StatusDetails](svc.status.status).
			WhereKeys(ontology.IDsToString(device.OntologyIDsFromDevices(res.Devices))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			res.Devices[i].Status = &stat
		}
	}

	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: device.OntologyIDsFromDevices(res.Devices),
	}); err != nil {
		return DeviceRetrieveResponse{}, err
	}
	if retErr != nil && req.IgnoreNotFound {
		retErr = errors.Skip(retErr, query.NotFound)
	}
	return res, retErr
}

type DeviceDeleteRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (svc *DeviceService) Delete(ctx context.Context, req DeviceDeleteRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: device.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.device.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}
