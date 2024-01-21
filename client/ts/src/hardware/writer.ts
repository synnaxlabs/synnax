// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { z } from "zod";

export const rackKeyZ = z.number();
export const moduleKeyZ = z.bigint().or(z.number().transform((n) => BigInt(n)));
export const deviceKeyZ = z.string();

export const rackZ = z.object({
  key: rackKeyZ,
  name: z.string(),
});

export const moduleZ = z.object({
  key: moduleKeyZ,
  name: z.string(),
  type: z.string(),
  config: z.string(),
});

export const deviceZ = z.object({
  key: deviceKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  properties: z.string(),
})

export const newRackZ = rackZ.partial({ key: true });
export const newModuleZ = moduleZ
  .omit({ key: true })
  .extend({ key: moduleKeyZ.transform((k) => k.toString()).optional() });

const createRackReqZ = z.object({
  racks: newRackZ.array(),
});

const createRackResZ = z.object({
  racks: rackZ.array(),
});

const deleteRackReqZ = z.object({
  keys: rackKeyZ.array(),
});

const deleteRackResZ = z.object({});

const createModuleReqZ = z.object({
  modules: newModuleZ.array(),
});

const createModuleResZ = z.object({
  modules: moduleZ.array(),
});

const deleteModuleReqZ = z.object({
  keys: moduleKeyZ.array(),
});

const deleteModuleResZ = z.object({});

const createDeviceReqZ = z.object({
  devices: deviceZ.array(),
});

const createDeviceResZ = z.object({
  devices: deviceZ.array(),
});

const deleteDeviceReqZ = z.object({
  keys: deviceKeyZ.array(),
});

const deleteDeviceResZ = z.object({});

const CREATE_RACK_ENDPOINT = "/hardware/rack/create";
const DELETE_RACK_ENDPOINT = "/hardware/rack/delete";
const CREATE_MODULE_ENDPOINT = "/hardware/module/create";
const DELETE_MODULE_ENDPOINT = "/hardware/module/delete";
const CREATE_DEVICE_ENDPOINT = "/hardware/device/create";
const DELETE_DEVICE_ENDPOINT = "/hardware/device/delete";

export type NewRackPayload = z.infer<typeof newRackZ>;
export type RackPayload = z.infer<typeof rackZ>;
export type NewModulePayload = z.input<typeof newModuleZ>;
export type ModulePayload = z.infer<typeof moduleZ>;
export type DevicePayload = z.infer<typeof deviceZ>;

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async createRack(racks: NewRackPayload[]): Promise<RackPayload[]> {
    const res = await sendRequired<typeof createRackReqZ, typeof createRackResZ>(
      this.client,
      CREATE_RACK_ENDPOINT,
      { racks },
      createRackResZ,
    );
    return res.racks;
  }

  async deleteRack(keys: number[]): Promise<void> {
    await sendRequired<typeof deleteRackReqZ, typeof deleteRackResZ>(
      this.client,
      DELETE_RACK_ENDPOINT,
      { keys },
      deleteRackResZ,
    );
  }

  async createModule(modules: NewModulePayload[]): Promise<ModulePayload[]> {
    const res = await sendRequired<typeof createModuleReqZ, typeof createModuleResZ>(
      this.client,
      CREATE_MODULE_ENDPOINT,
      createModuleReqZ.parse({ modules }),
      createModuleResZ,
    );
    return res.modules;
  }

  async deleteModule(keys: bigint[]): Promise<void> {
    await sendRequired<typeof deleteModuleReqZ, typeof deleteModuleResZ>(
      this.client,
      DELETE_MODULE_ENDPOINT,
      { keys },
      deleteModuleResZ,
    );
  }

  async createDevice(devices: DevicePayload[]): Promise<DevicePayload[]> {
    const res = await sendRequired<typeof createDeviceReqZ, typeof createDeviceResZ>(
      this.client,
      CREATE_DEVICE_ENDPOINT,
      createDeviceReqZ.parse({ devices }),
      createDeviceResZ,
    );
    return res.devices;
  }

  async deleteDevice(keys: string[]): Promise<void> {
    await sendRequired<typeof deleteDeviceReqZ, typeof deleteDeviceResZ>(
      this.client,
      DELETE_DEVICE_ENDPOINT,
      { keys },
      deleteDeviceResZ,
    );
  }
}
