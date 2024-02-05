// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

export const rackKeyZ = z.number();
export const taskKeyZ = z.bigint().or(z.number()).transform((k) => k.toString())
export const deviceKeyZ = z.string();

export const rackZ = z.object({
  key: rackKeyZ,
  name: z.string(),
});

export const taskZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  type: z.string(),
  config: z.record(z.unknown()),
});

export type Task<
  T extends string = string, 
  C extends UnknownRecord = UnknownRecord
> = Omit<z.infer<typeof taskZ>, "config" | "type"> & { type: T, config: C };

export const deviceZ = z.object({
  key: deviceKeyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  properties: z.string(),
})

export const newRackZ = rackZ.partial({ key: true });
export const newTaskZ = taskZ
  .omit({ key: true })
  .extend({ key: taskKeyZ.transform((k) => k.toString()).optional(), config: z.unknown().transform((c) => JSON.stringify(c)) });

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

const createTaskReqZ = z.object({
  tasks: newTaskZ.array(),
});

const createTaskResZ = z.object({
  tasks: taskZ.array(),
});

const deleteTaskReqZ = z.object({
  keys: taskKeyZ.array(),
});

const deleteTaskResZ = z.object({});

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
const CREATE_TASK_ENDPOINT = "/hardware/task/create";
const DELETE_TASK_ENDPOINT = "/hardware/task/delete";
const CREATE_DEVICE_ENDPOINT = "/hardware/device/create";
const DELETE_DEVICE_ENDPOINT = "/hardware/device/delete";

export type NewRackPayload = z.infer<typeof newRackZ>;
export type RackPayload = z.infer<typeof rackZ>;
export type NewTaskPayload = z.input<typeof newTaskZ>;
export type TaskPayload = z.infer<typeof taskZ>;
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

  async createTask(tasks: NewTaskPayload[]): Promise<TaskPayload[]> {
    const res = await sendRequired<typeof createTaskReqZ, typeof createTaskResZ>(
      this.client,
      CREATE_TASK_ENDPOINT,
      {tasks: tasks.map((t) => ({...t, config: JSON.stringify(t.config)}))},
      createTaskResZ,
    );
    return res.tasks;
  }

  async deleteTask(keys: bigint[]): Promise<void> {
    await sendRequired<typeof deleteTaskReqZ, typeof deleteTaskResZ>(
      this.client,
      DELETE_TASK_ENDPOINT,
      { keys },
      deleteTaskResZ,
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
