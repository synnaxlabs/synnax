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
	"github.com/synnaxlabs/synnax/pkg/hardware/module"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
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
type Module = module.Module
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

type HardwareCreateModuleRequest struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

type HardwareCreateModuleResponse struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

func (svc *HardwareService) CreateModule(ctx context.Context, req HardwareCreateModuleRequest) (res HardwareCreateModuleResponse, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Module.NewWriter(tx)
		for i, m := range req.Modules {
			if err := w.Create(ctx, &m); err != nil {
				return err
			}
			req.Modules[i] = m
		}
		res.Modules = req.Modules
		return nil
	})
}

type HardwareRetrieveModuleRequest struct {
	Rack rack.Key
	Keys []module.Key `json:"keys" msgpack:"keys"`
}

type HardwareRetrieveModuleResponse struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

func (svc *HardwareService) RetrieveModule(ctx context.Context, req HardwareRetrieveModuleRequest) (res HardwareRetrieveModuleResponse, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		r := svc.internal.Module.NewRetrieve()
		if len(req.Keys) > 0 {
			r = r.WhereKeys(req.Keys...)
		}
		if req.Rack.IsValid() {
			r = r.WhereRack(req.Rack)
		}
		return r.Entries(&res.Modules).Exec(ctx, tx)
	})
}

type HardwareDeleteModuleRequest struct {
	Keys []module.Key `json:"keys" msgpack:"keys"`
}

func (svc *HardwareService) DeleteModule(ctx context.Context, req HardwareDeleteModuleRequest) (res types.Nil, _ error) {
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.internal.Module.NewWriter(tx)
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
