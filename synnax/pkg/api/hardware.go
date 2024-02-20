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
	"github.com/synnaxlabs/synnax/pkg/hardware"
	"github.com/synnaxlabs/synnax/pkg/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type HardwareService struct {
	dbProvider
	internal *hardware.Service
}

func NewHardwareService(p Provider) *HardwareService {
	return &HardwareService{
		dbProvider: p.db,
		internal:   p.Config.Hardware,
	}
}

type Rack = rack.Rack
type Task = task.Task
type Device = device.Device

type HardwareCreateRackRequest struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

type HardwareCreateRackResponse struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

func (svc *HardwareService) CreateRack(ctx context.Context, req HardwareCreateRackRequest) (res HardwareCreateRackResponse, _ error) {
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

type HardwareRetrieveRackRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

type HardwareRetrieveRackResponse struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

func (svc *HardwareService) RetrieveRack(ctx context.Context, req HardwareRetrieveRackRequest) (res HardwareRetrieveRackResponse, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		err := svc.internal.Rack.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Racks).Exec(ctx, tx)
		return err
	})
}

type HardwareDeleteRackRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteRack(ctx context.Context, req HardwareDeleteRackRequest) (res types.Nil, _ error) {
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

type HardwareCreateTaskRequest struct {
	Tasks []task.Task `json:"tasks" msgpack:"tasks"`
}

type HardwareCreateTaskResponse struct {
	Tasks []task.Task `json:"tasks" msgpack:"tasks"`
}

func (svc *HardwareService) CreateTask(ctx context.Context, req HardwareCreateTaskRequest) (res HardwareCreateTaskResponse, _ error) {
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

type HardwareRetrieveTaskRequest struct {
	Rack rack.Key
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

type HardwareRetrieveTaskResponse struct {
	Tasks []task.Task `json:"tasks" msgpack:"tasks"`
}

func (svc *HardwareService) RetrieveTask(ctx context.Context, req HardwareRetrieveTaskRequest) (res HardwareRetrieveTaskResponse, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		r := svc.internal.Task.NewRetrieve()
		if len(req.Keys) > 0 {
			r = r.WhereKeys(req.Keys...)
		}
		if req.Rack.IsValid() {
			r = r.WhereRack(req.Rack)
		}
		res.Tasks = make([]task.Task, 0, len(req.Keys))
		return r.Entries(&res.Tasks).Exec(ctx, tx)
	})
}

type HardwareDeleteTaskRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteTask(ctx context.Context, req HardwareDeleteTaskRequest) (res types.Nil, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Task.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}

type HardwareCreateDeviceRequest struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

type HardwareCreateDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) CreateDevice(ctx context.Context, req HardwareCreateDeviceRequest) (res HardwareCreateDeviceResponse, _ error) {
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
	Keys []string `json:"keys" msgpack:"keys"`
}

type HardwareRetrieveDeviceResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (svc *HardwareService) RetrieveDevice(ctx context.Context, req HardwareRetrieveDeviceRequest) (res HardwareRetrieveDeviceResponse, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.internal.Device.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Devices).Exec(ctx, tx)
	})
}

type HardwareDeleteDeviceRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteDevice(ctx context.Context, req HardwareDeleteDeviceRequest) (res types.Nil, _ error) {
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
