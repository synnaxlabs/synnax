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

export const SET_CHANNEL_NAME = "sy_device_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";

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
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const retrieveResZ = z.object({ devices: array.nullableZ(deviceZ) });

const singleRetrieveArgsZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({
    keys: [key],
    includeStatus,
  }));

export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(args: RetrieveSingleParams): Promise<Device<Properties, Make, Model>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(args: RetrieveMultipleParams): Promise<Array<Device<Properties, Make, Model>>>;

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
      "/hardware/device/retrieve",
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
      "/hardware/device/create",
      { devices: array.toArray(devices) },
      createReqZ,
      createResZ,
    );
    const created = res.devices as Device<Properties, Make, Model>[];
    return isSingle ? created[0] : created;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/hardware/device/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "device", key });
