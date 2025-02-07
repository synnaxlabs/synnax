// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { type framer } from "@/framer";
import {
  type Device,
  deviceZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
} from "@/hardware/device/payload";
import { ontology } from "@/ontology";
import { signals } from "@/signals";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const SET_CHANNEL_NAME = "sy_device_set";
const DELETE_CHANNEL_NAME = "sy_device_delete";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";
const CREATE_ENDPOINT = "/hardware/device/create";
const DELETE_ENDPOINT = "/hardware/device/delete";

const createReqZ = z.object({ devices: newZ.array() });

const createResZ = z.object({ devices: deviceZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

const retrieveReqZ = z.object({
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
});

interface RetrieveRequest extends z.input<typeof retrieveReqZ> {}

export interface RetrieveOptions
  extends Pick<RetrieveRequest, "limit" | "offset" | "makes"> {}

interface PageOptions extends Pick<RetrieveOptions, "makes"> {}

const retrieveResZ = z.object({ devices: nullableArrayZ(deviceZ) });

export class Client implements AsyncTermSearcher<string, Key, Device> {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async retrieve<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(key: string, options?: RetrieveOptions): Promise<Device<P, MK, MO>>;

  async retrieve<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(keys: string[], options?: RetrieveOptions): Promise<Array<Device<P, MK, MO>>>;

  async retrieve<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(
    keys: string | string[],
    options?: RetrieveOptions,
  ): Promise<Device<P, MK, MO> | Array<Device<P, MK, MO>>> {
    const isSingle = !Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys), ...options },
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Device", keys, res.devices, isSingle);
    return isSingle
      ? (res.devices[0] as Device<P, MK, MO>)
      : (res.devices as Array<Device<P, MK, MO>>);
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

  async create<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(device: New<P, MK>): Promise<Device<P, MK, MO>>;
  async create<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(devices: New<P, MK>[]): Promise<Device<P, MK, MO>[]>;
  async create<
    P extends UnknownRecord = UnknownRecord,
    MK extends string = string,
    MO extends string = string,
  >(
    devices: New<P, MK> | New<P, MK>[],
  ): Promise<Device<P, MK, MO> | Device<P, MK, MO>[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { devices: toArray(devices) },
      createReqZ,
      createResZ,
    );
    return isSingle
      ? (res.devices[0] as Device<P, MK, MO>)
      : (res.devices as Device<P, MK, MO>[]);
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
      SET_CHANNEL_NAME,
      DELETE_CHANNEL_NAME,
      decodeDeviceChanges,
    );
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, Key, Device> {
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

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
