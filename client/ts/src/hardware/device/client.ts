// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, type record } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Device,
  deviceZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
} from "@/hardware/device/payload";
import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import { type ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

export const SET_CHANNEL_NAME = "sy_device_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";
export const STATUS_CHANNEL_NAME = "sy_device_status";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";
const CREATE_ENDPOINT = "/hardware/device/create";
const DELETE_ENDPOINT = "/hardware/device/delete";

const createReqZ = z.object({ devices: newZ.array() });

const createResZ = z.object({ devices: deviceZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
  models: z.string().array().optional(),
  locations: z.string().array().optional(),
  racks: rackKeyZ.array().optional(),
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  ignoreNotFound: z.boolean().optional(),
  includeStatus: z.boolean().optional(),
});

interface RetrieveRequest extends z.input<typeof retrieveReqZ> {}

export interface RetrieveOptions
  extends Pick<
    RetrieveRequest,
    "limit" | "offset" | "makes" | "ignoreNotFound" | "includeStatus"
  > {}

const retrieveResZ = z.object({ devices: nullableArrayZ(deviceZ) });

export class Client {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(key: string, options?: RetrieveOptions): Promise<Device<Properties, Make, Model>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(
    keys: string[],
    options?: RetrieveOptions,
  ): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(request: RetrieveRequest): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(
    keys: string | string[] | RetrieveRequest,
    options?: RetrieveOptions,
  ): Promise<Device<Properties, Make, Model> | Array<Device<Properties, Make, Model>>> {
    let request: RetrieveRequest;
    if (Array.isArray(keys)) request = { keys: array.toArray(keys), ...options };
    else if (typeof keys === "object") request = keys;
    else request = { keys: [keys], ...options };
    const isSingle = !Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      request,
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Device", keys, res.devices, isSingle);
    return isSingle
      ? (res.devices[0] as Device<Properties, Make, Model>)
      : (res.devices as Array<Device<Properties, Make, Model>>);
  }

  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(device: New<Properties, Make>): Promise<Device<Properties, Make, Model>>;
  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(devices: New<Properties, Make>[]): Promise<Device<Properties, Make, Model>[]>;
  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(
    devices: New<Properties, Make> | New<Properties, Make>[],
  ): Promise<Device<Properties, Make, Model> | Device<Properties, Make, Model>[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { devices: array.toArray(devices) },
      createReqZ,
      createResZ,
    );
    return isSingle
      ? (res.devices[0] as Device<Properties, Make, Model>)
      : (res.devices as Device<Properties, Make, Model>[]);
  }

  async delete(keys: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });
