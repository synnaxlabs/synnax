// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { binary } from "@synnaxlabs/x/binary";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { type framer } from "@/framer";
import { rackKeyZ } from "@/hardware/rack/client";
import { signals } from "@/signals";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
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
  configured: z.boolean().optional(),
  properties: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return binary.JSON_CODEC.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export type Device<P extends UnknownRecord = UnknownRecord> = Omit<
  z.output<typeof deviceZ>,
  "properties"
> & { properties: P };

export type DeviceKey = z.infer<typeof deviceKeyZ>;

export const newDeviceZ = deviceZ.extend({
  properties: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});

export type NewDevice<P extends UnknownRecord = UnknownRecord> = Omit<
  z.input<typeof newDeviceZ>,
  "properties"
> & { properties: P };

const createReqZ = z.object({ devices: newDeviceZ.array() });

const createResZ = z.object({ devices: deviceZ.array() });

const deleteReqZ = z.object({ keys: deviceKeyZ.array() });

const deleteResZ = z.object({});

const retrieveReqZ = z.object({
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  keys: deviceKeyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
});

type RetrieveRequest = z.input<typeof retrieveReqZ>;

export type RetrieveOptions = Pick<RetrieveRequest, "limit" | "offset" | "makes">;

type PageOptions = Pick<RetrieveOptions, "makes">;

const retrieveResZ = z.object({ devices: nullableArrayZ(deviceZ) });

export class Client implements AsyncTermSearcher<string, DeviceKey, Device> {
  readonly type = "device";
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async retrieve<P extends UnknownRecord = UnknownRecord>(
    key: string,
    options?: RetrieveOptions,
  ): Promise<Device<P>>;

  async retrieve<P extends UnknownRecord = UnknownRecord>(
    keys: string[],
    options?: RetrieveOptions,
  ): Promise<Array<Device<P>>>;

  async retrieve<P extends UnknownRecord = UnknownRecord>(
    keys: string | string[],
    options?: RetrieveOptions,
  ): Promise<Device<P> | Array<Device<P>>> {
    const isSingle = !Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys), ...options },
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Device", keys, res.devices, isSingle);
    return (isSingle ? res.devices[0] : res.devices) as Device<P> | Array<Device<P>>;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Device[]> {
    return (
      await sendRequired(
        this.client,
        RETRIEVE_ENDPOINT,
        { search: term, ...options },
        retrieveReqZ,
        retrieveResZ,
      )
    ).devices;
  }

  async page(offset: number, limit: number, options?: PageOptions): Promise<Device[]> {
    return (
      await sendRequired(
        this.client,
        RETRIEVE_ENDPOINT,
        { offset, limit, ...options },
        retrieveReqZ,
        retrieveResZ,
      )
    ).devices;
  }

  async create(device: NewDevice): Promise<Device>;

  async create(devices: NewDevice[]): Promise<Device[]>;

  async create(devices: NewDevice | NewDevice[]): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { devices: toArray(devices) },
      createReqZ,
      createResZ,
    );
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async openDeviceTracker(): Promise<signals.Observable<string, Device>> {
    return await signals.openObservable<string, Device>(
      this.frameClient,
      DEVICE_SET_NAME,
      DEVICE_DELETE_NAME,
      decodeDeviceChanges,
    );
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, DeviceKey, Device> {
    return {
      type: this.type,
      search: async (term: string) => await this.search(term, options),
      retrieve: async (keys: string[]) => await this.retrieve(keys, options),
      page: async (offset: number, limit: number) =>
        await this.page(offset, limit, options),
    };
  }
}

const decodeDeviceChanges: signals.Decoder<string, Device> = (variant, data) => {
  if (variant === "delete")
    return data.toStrings().map((k) => ({ variant, key: k, value: undefined }));
  return data.parseJSON(deviceZ).map((d) => ({ variant, key: d.key, value: d }));
};
