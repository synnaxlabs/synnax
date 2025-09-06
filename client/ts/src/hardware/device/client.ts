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

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
  models: z.string().array().optional(),
  locations: z.string().array().optional(),
  racks: rackKeyZ.array().optional(),
  searchTerm: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  includeStatus: z.boolean().optional(),
});
const retrieveResZ = z.object({ devices: nullableArrayZ(deviceZ) });

const singleRetrieveArgsZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({
    keys: [key],
    includeStatus,
  }));

export type SingleRetrieveArgs = z.input<typeof singleRetrieveArgsZ>;
export type MultiRetrieveArgs = z.input<typeof retrieveRequestZ>;

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  readonly type = "device";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(args: SingleRetrieveArgs): Promise<Device<Properties, Make, Model>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(args: MultiRetrieveArgs): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(
    args: RetrieveArgs,
  ): Promise<Device<Properties, Make, Model> | Array<Device<Properties, Make, Model>>> {
    const isSingle = typeof args === "object" && "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Device", args, res.devices, isSingle);
    const devices = res.devices as Device<Properties, Make, Model>[];
    return isSingle ? devices[0] : devices;
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
    const created = res.devices as Device<Properties, Make, Model>[];
    return isSingle ? created[0] : created;
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

export const ontologyID = (key: Key): ontology.ID => ({ type: "device", key });
