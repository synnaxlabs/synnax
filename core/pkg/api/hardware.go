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
	"github.com/synnaxlabs/synnax/pkg/service/hardware"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type HardwareService struct {
	dbProvider
	accessProvider
	internal *hardware.Service
}

func NewHardwareService(p Provider) *HardwareService {
	return &HardwareService{
		dbProvider:     p.db,
		internal:       p.Service.Hardware,
		accessProvider: p.access,
	}
}

type (
	Rack   = rack.Rack
	Task   = task.Task
	Device = device.Device
)

type (
	HardwareCreateRackRequest struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
	HardwareCreateRackResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (svc *HardwareService) CreateRack(ctx context.Context, req HardwareCreateRackRequest) (res HardwareCreateRackResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: rack.OntologyIDsFromRacks(req.Racks),
	}); err != nil {
		return res, err
	}
	if err := svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Rack.NewWriter(tx)
		for i, r := range req.Racks {
			if err := w.Create(ctx, &r); err != nil {
				return err
			}
			req.Racks[i] = r
		}
		res.Racks = req.Racks
		return nil
	}); err != nil {
		return HardwareCreateRackResponse{}, err
	}
	return res, nil
}

type (
	HardwareRetrieveRackRequest struct {
		Keys          []rack.Key `json:"keys" msgpack:"keys"`
		Names         []string   `json:"names" msgpack:"names"`
		SearchTerm    string     `json:"search_term" msgpack:"search_term"`
		Embedded      *bool      `json:"embedded" msgpack:"embedded"`
		HostIsNode    *bool      `json:"host_is_node" msgpack:"host_is_node"`
		Limit         int        `json:"limit" msgpack:"limit"`
		Offset        int        `json:"offset" msgpack:"offset"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
	}
	HardwareRetrieveRackResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (svc *HardwareService) RetrieveRack(ctx context.Context, req HardwareRetrieveRackRequest) (res HardwareRetrieveRackResponse, _ error) {
	var (
		hasSearch = len(req.SearchTerm) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	resRacks := make([]rack.Rack, 0, len(req.Keys)+len(req.Names))
	q := svc.internal.Rack.NewRetrieve()
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
		q = q.WhereNodeIsHost()
	}
	if err := q.Entries(&resRacks).Exec(ctx, nil); err != nil {
		return res, err
	}

	if req.IncludeStatus {
		for i := range resRacks {
			if s, ok := svc.internal.State.GetRack(ctx, resRacks[i].Key); ok {
				resRacks[i].Status = &s.Status
			}
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

type HardwareDeleteRackRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

func embeddedGuard(_ gorp.Context, r Rack) error {
	if !r.Embedded {
		return nil
	}
	return errors.Wrapf(validate.Error, "cannot delete embedded rack")
}

func (svc *HardwareService) DeleteRack(
	ctx context.Context,
	req HardwareDeleteRackRequest,
) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: rack.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		exists, err := svc.internal.Device.NewRetrieve().WhereRacks(req.Keys...).Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(validate.Error, "cannot delete rack when devices are still attached")
		}
		exists, err = svc.internal.Task.NewRetrieve().WhereInternal(false, gorp.Required()).WhereRacks(req.Keys...).Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(validate.Error, "cannot delete rack when tasks are still attached")
		}
		w := svc.internal.Rack.NewWriter(tx)
		for _, k := range req.Keys {
			if err = w.DeleteGuard(ctx, k, embeddedGuard); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	HardwareCreateTaskRequest struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
	HardwareCreateTaskResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *HardwareService) CreateTask(ctx context.Context, req HardwareCreateTaskRequest) (res HardwareCreateTaskResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: task.OntologyIDsFromTasks(req.Tasks),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Task.NewWriter(tx)
		for i, m := range req.Tasks {
			if err := w.Create(ctx, &m); err != nil {
				return err
			}
			req.Tasks[i] = m
		}
		res.Tasks = req.Tasks
		return nil
	})
}

type (
	HardwareRetrieveTaskRequest struct {
		Rack          rack.Key   `json:"rack" msgpack:"rack"`
		Keys          []task.Key `json:"keys" msgpack:"keys"`
		Names         []string   `json:"names" msgpack:"names"`
		Types         []string   `json:"types" msgpack:"types"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
		Internal      *bool      `json:"internal" msgpack:"internal"`
		Snapshot      *bool      `json:"snapshot" msgpack:"snapshot"`
		SearchTerm    string     `json:"search_term" msgpack:"search_term"`
		Limit         int        `json:"limit" msgpack:"limit"`
		Offset        int        `json:"offset" msgpack:"offset"`
	}
	HardwareRetrieveTaskResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *HardwareService) RetrieveTask(
	ctx context.Context,
	req HardwareRetrieveTaskRequest,
) (res HardwareRetrieveTaskResponse, _ error) {
	var (
		hasSearch = len(req.SearchTerm) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasTypes  = len(req.Types) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	q := svc.internal.Task.NewRetrieve()
	if req.Internal != nil {
		q = q.WhereInternal(*req.Internal, gorp.Required())
	}
	if req.Snapshot != nil {
		q = q.WhereSnapshot(*req.Snapshot, gorp.Required())
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasTypes {
		q = q.WhereTypes(req.Types...)
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
	if !req.Rack.IsZero() {
		q = q.WhereRacks(req.Rack)
	}
	err := q.Entries(&res.Tasks).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if req.IncludeStatus {
		for i := range res.Tasks {
			s, ok := svc.internal.State.GetTask(ctx, res.Tasks[i].Key)
			if ok {
				res.Tasks[i].Status = &s
			}
		}
	}
	if err = svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: task.OntologyIDsFromTasks(res.Tasks),
	}); err != nil {
		return HardwareRetrieveTaskResponse{}, err
	}
	return res, nil
}

type HardwareDeleteTaskRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteTask(
	ctx context.Context,
	req HardwareDeleteTaskRequest,
) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: task.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Task.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k, false); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	HardwareCopyTaskRequest struct {
		Key      task.Key `json:"key" msgpack:"key"`
		Name     string   `json:"name" msgpack:"name"`
		Snapshot bool     `json:"snapshot" msgpack:"snapshot"`
	}
	HardwareCopyTaskResponse struct {
		Task task.Task `json:"task" msgpack:"task"`
	}
)

func (svc *HardwareService) CopyTask(ctx context.Context, req HardwareCopyTaskRequest) (res HardwareCopyTaskResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: []ontology.ID{task.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	err := svc.WithTx(ctx, func(tx gorp.Tx) (err error) {
		res.Task, err = svc.internal.Task.NewWriter(tx).Copy(
			ctx,
			req.Key,
			req.Name,
			req.Snapshot,
		)
		return err
	})
	if err != nil {
		return HardwareCopyTaskResponse{}, err
	}
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{task.OntologyID(res.Task.Key)},
	}); err != nil {
		return HardwareCopyTaskResponse{}, err
	}
	return res, nil
}

type HardwareCreateDeviceRequest struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

type HardwareCreateDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) CreateDevice(ctx context.Context, req HardwareCreateDeviceRequest) (res HardwareCreateDeviceResponse, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: device.OntologyIDsFromDevices(req.Devices),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Device.NewWriter(tx)
		for _, d := range req.Devices {
			if err := w.Create(ctx, d); err != nil {
				return err
			}
		}
		res.Devices = req.Devices
		return nil
	})
}

type HardwareRetrieveDeviceRequest struct {
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

type HardwareRetrieveDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) RetrieveDevice(ctx context.Context, req HardwareRetrieveDeviceRequest) (res HardwareRetrieveDeviceResponse, _ error) {
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
	q := svc.internal.Device.NewRetrieve()
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
		for i := range res.Devices {
			if s, ok := svc.internal.State.GetDevice(ctx, res.Devices[i].Key); ok {
				res.Devices[i].Status = &s
			}
		}
	}
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: device.OntologyIDsFromDevices(res.Devices),
	}); err != nil {
		return HardwareRetrieveDeviceResponse{}, err
	}
	if retErr != nil && req.IgnoreNotFound {
		retErr = errors.Skip(retErr, query.NotFound)
	}
	return res, retErr
}

type HardwareDeleteDeviceRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteDevice(ctx context.Context, req HardwareDeleteDeviceRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: device.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Device.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}
