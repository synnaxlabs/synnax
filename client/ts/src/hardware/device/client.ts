// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { type framer } from "@/framer";
import { rackKeyZ } from "@/hardware/rack/client";
import { signals } from "@/signals";
import { nullableArrayZ } from "@/util/zod";

const DEVICE_SET_NAME = "sy_device_set";
const DEVICE_DELETE_NAME = "sy_device_delete";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";

const CREATE_ENDPOINT = "/hardware/device/create";
const DELETE_ENDPOINT = "/hardware/device/delete";

export const deviceKeyZ = z.string();

export const deviceZ = z.object({
  key: deviceKeyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  properties: z.string(),
});

export type Device = z.infer<typeof deviceZ>;
export type DeviceKey = z.infer<typeof deviceKeyZ>;

const createReqZ = z.object({
  devices: deviceZ.array(),
});

const createResZ = z.object({
  devices: deviceZ.array(),
});

const deleteReqZ = z.object({
  keys: deviceKeyZ.array(),
});

const deleteResZ = z.object({});

const retrieveDeviceReqZ = z.object({
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  keys: deviceKeyZ.array().optional(),
});

const retrieveDeviceResZ = z.object({
  devices: nullableArrayZ(deviceZ),
});

export class Client implements AsyncTermSearcher<string, DeviceKey, Device> {
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async retrieve(key: string): Promise<Device>;

  async retrieve(keys: string[]): Promise<Device[]>;

  async retrieve(keys: string | string[]): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(keys);
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveDeviceReqZ,
      retrieveDeviceResZ,
    );
    return isSingle ? res.devices[0] : res.devices;
  }

  async search(term: string): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: [term] },
      retrieveDeviceReqZ,
      retrieveDeviceResZ,
    );
    return res.devices;
  }

  async page(offset: number, limit: number): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveDeviceReqZ,
      retrieveDeviceResZ,
    );
    return res.devices;
  }

  async create(device: Device): Promise<Device>;

  async create(devices: Device | Device[]): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { devices: toArray(devices) },
      createReqZ,
      createResZ,
    );
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: string | string[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async openDeviceTracker(): Promise<signals.Observable<string, Device>> {
    return await signals.Observable.open<string, Device>(
      this.frameClient,
      DEVICE_SET_NAME,
      DEVICE_DELETE_NAME,
      decodeDeviceChanges,
    );
  }
}

const decodeDeviceChanges: signals.Decoder<string, Device> = (variant, data) => {
  if (variant === "delete")
    return data.toStrings().map((k) => ({ variant, key: k, value: undefined }));
  return data.parseJSON(deviceZ).map((d) => ({ variant, key: d.key, value: d }));
};
