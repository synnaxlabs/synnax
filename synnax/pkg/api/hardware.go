/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package api

import (
	"context"
	access2 "github.com/synnaxlabs/synnax/pkg/service/access"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/hardware"
	"github.com/synnaxlabs/synnax/pkg/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/gorp"
)

type HardwareService struct {
	dbProvider
	accessProvider
	internal *hardware.Service
}

func NewHardwareService(p Provider) *HardwareService {
	return &HardwareService{
		dbProvider:     p.db,
		internal:       p.Config.Hardware,
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
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Create,
		Objects: []ontology.ID{rack.OntologyID(0)},
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Rack.NewWriter(tx)
		for i, r := range req.Racks {
			if err := w.Create(ctx, &r); err != nil {
				return err
			}
			req.Racks[i] = r
		}
		res.Racks = req.Racks
		return nil
	})
}

type (
	HardwareRetrieveRackRequest struct {
		Keys   []rack.Key `json:"keys" msgpack:"keys"`
		Names  []string   `json:"names" msgpack:"names"`
		Search string     `json:"search" msgpack:"search"`
		Limit  int        `json:"limit" msgpack:"limit"`
		Offset int        `json:"offset" msgpack:"offset"`
	}
	HardwareRetrieveRackResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (svc *HardwareService) RetrieveRack(ctx context.Context, req HardwareRetrieveRackRequest) (res HardwareRetrieveRackResponse, _ error) {
	var (
		hasSearch = len(req.Search) > 0
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
		q = q.Search(req.Search)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if err := q.Entries(&resRacks).Exec(ctx, nil); err != nil {
		return res, err
	}
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Retrieve,
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

func (svc *HardwareService) DeleteRack(ctx context.Context, req HardwareDeleteRackRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Delete,
		Objects: rack.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Rack.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
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
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Create,
		Objects: []ontology.ID{task.OntologyID(0)},
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
		Rack         rack.Key
		Keys         []task.Key `json:"keys" msgpack:"keys"`
		Names        []string   `json:"names" msgpack:"names"`
		Types        []string   `json:"types" msgpack:"types"`
		IncludeState bool       `json:"include_state" msgpack:"include_state"`
		Search       string     `json:"search" msgpack:"search"`
		Limit        int        `json:"limit" msgpack:"limit"`
		Offset       int        `json:"offset" msgpack:"offset"`
	}
	HardwareRetrieveTaskResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *HardwareService) RetrieveTask(ctx context.Context, req HardwareRetrieveTaskRequest) (res HardwareRetrieveTaskResponse, _ error) {
	var (
		hasSearch = len(req.Search) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasTypes  = len(req.Types) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	q := svc.internal.Task.NewRetrieve()
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
		q = q.Search(req.Search)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if req.Rack.IsValid() && len(req.Names) == 0 {
		q = q.WhereRack(req.Rack)
	}
	err := q.Entries(&res.Tasks).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if req.IncludeState {
		for i := range res.Tasks {
			s, ok := svc.internal.State.GetTask(ctx, res.Tasks[i].Key)
			if ok {
				res.Tasks[i].State = &s
			}
		}
	}
	if err = svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Retrieve,
		Objects: task.OntologyIDsFromTasks(res.Tasks),
	}); err != nil {
		return HardwareRetrieveTaskResponse{}, err
	}
	return res, nil
}

type HardwareDeleteTaskRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteTask(ctx context.Context, req HardwareDeleteTaskRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Delete,
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
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Create,
		Objects: []ontology.ID{task.OntologyID(0)},
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
	return res, err
}

type HardwareCreateDeviceRequest struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

type HardwareCreateDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) CreateDevice(ctx context.Context, req HardwareCreateDeviceRequest) (res HardwareCreateDeviceResponse, _ error) {
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Create,
		Objects: []ontology.ID{device.OntologyID("")},
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
	Keys      []string `json:"keys" msgpack:"keys"`
	Names     []string `json:"names" msgpack:"names"`
	Makes     []string `json:"makes" msgpack:"makes"`
	Models    []string `json:"models" msgpack:"models"`
	Locations []string `json:"locations" msgpack:"locations"`
	Search    string   `json:"search" msgpack:"search"`
	Limit     int      `json:"limit" msgpack:"limit"`
	Offset    int      `json:"offset" msgpack:"offset"`
}

type HardwareRetrieveDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) RetrieveDevice(ctx context.Context, req HardwareRetrieveDeviceRequest) (res HardwareRetrieveDeviceResponse, _ error) {
	var (
		hasSearch    = len(req.Search) > 0
		hasKeys      = len(req.Keys) > 0
		hasNames     = len(req.Names) > 0
		hasMakes     = len(req.Makes) > 0
		hasLimit     = req.Limit > 0
		hasOffset    = req.Offset > 0
		hasLocations = len(req.Locations) > 0
		hasModels    = len(req.Models) > 0
	)
	q := svc.internal.Device.NewRetrieve()
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.Search)
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
	err := q.Entries(&res.Devices).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if err = svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Retrieve,
		Objects: device.OntologyIDsFromDevices(res.Devices),
	}); err != nil {
		return HardwareRetrieveDeviceResponse{}, err
	}
	return res, nil
}

type HardwareDeleteDeviceRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteDevice(ctx context.Context, req HardwareDeleteDeviceRequest) (res types.Nil, _ error) {
	if err := svc.access.Enforce(ctx, access2.Request{
		Subject: getSubject(ctx),
		Action:  access2.Delete,
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
