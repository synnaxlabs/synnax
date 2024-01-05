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
	"github.com/synnaxlabs/synnax/pkg/device"
	"github.com/synnaxlabs/synnax/pkg/device/module"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type DeviceService struct {
	dbProvider
	internal *device.Service
}

func NewDeviceService(p Provider) *DeviceService {
	return &DeviceService{
		dbProvider: p.db,
		internal:   p.Config.Device,
	}
}

type Rack = rack.Rack
type Module = module.Module

type DeviceCreateRackRequest struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

type DeviceCreateRackResponse struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

func (d *DeviceService) CreateRack(ctx context.Context, req DeviceCreateRackRequest) (res DeviceCreateRackResponse, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		w := d.internal.Rack.NewWriter(tx)
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

type DeviceRetrieveRackRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

type DeviceRetrieveRackResponse struct {
	Racks []rack.Rack `json:"racks" msgpack:"racks"`
}

func (d *DeviceService) RetrieveRack(ctx context.Context, req DeviceRetrieveRackRequest) (res DeviceRetrieveRackResponse, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		return d.internal.Rack.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Racks).Exec(ctx, tx)
	})
}

type DeviceDeleteRackRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

func (d *DeviceService) DeleteRack(ctx context.Context, req DeviceDeleteRackRequest) (res types.Nil, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		w := d.internal.Rack.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}

type DeviceCreateModuleRequest struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

type DeviceCreateModuleResponse struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

func (d *DeviceService) CreateModule(ctx context.Context, req DeviceCreateModuleRequest) (res DeviceCreateModuleResponse, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		w := d.internal.Module.NewWriter(tx)
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

type DeviceRetrieveModuleRequest struct {
	Rack rack.Key
	Keys []module.Key `json:"keys" msgpack:"keys"`
}

type DeviceRetrieveModuleResponse struct {
	Modules []module.Module `json:"modules" msgpack:"modules"`
}

func (d *DeviceService) RetrieveModule(ctx context.Context, req DeviceRetrieveModuleRequest) (res DeviceRetrieveModuleResponse, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		r := d.internal.Module.NewRetrieve()
		if len(req.Keys) > 0 {
			r = r.WhereKeys(req.Keys...)
		}
		if req.Rack.IsValid() {
			r = r.WhereRack(req.Rack)
		}
		return r.Entries(&res.Modules).Exec(ctx, tx)
	})
}

type DeviceDeleteModuleRequest struct {
	Keys []module.Key `json:"keys" msgpack:"keys"`
}

func (d *DeviceService) DeleteModule(ctx context.Context, req DeviceDeleteModuleRequest) (res types.Nil, _ error) {
	return res, d.WithTx(ctx, func(tx gorp.Tx) error {
		w := d.internal.Module.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}
