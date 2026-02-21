// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, zod } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Device,
  type DeviceSchemas,
  deviceZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ontologyID,
} from "@/device/types.gen";
import { ontology } from "@/ontology";
import { keyZ as rackKeyZ } from "@/rack/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_device_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";

const createReqZ = <
  Properties extends z.ZodType = z.ZodType,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) => z.object({ devices: zod.toArray(newZ(schemas)) });

const createResZ = <
  Properties extends z.ZodType = z.ZodType,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) => z.object({ devices: deviceZ(schemas).array() });

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

const retrieveResZ = <
  Properties extends z.ZodType = z.ZodType,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) => z.object({ devices: array.nullishToEmpty(deviceZ(schemas)) });

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

type RetrieveSchemas<
  Properties extends z.ZodType,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> = { schemas: DeviceSchemas<Properties, Make, Model> };

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve<
    Properties extends z.ZodType,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    args: RetrieveSingleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async retrieve(args: RetrieveSingleParams): Promise<Device>;

  async retrieve<
    Properties extends z.ZodType,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    args: RetrieveMultipleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve(args: RetrieveMultipleParams): Promise<Array<Device>>;

  async retrieve(
    args: RetrieveArgs & { schemas?: DeviceSchemas },
  ): Promise<Device | Array<Device>> {
    const { schemas, ...rest } = args as RetrieveArgs & { schemas?: DeviceSchemas };
    const isSingle = typeof rest === "object" && "key" in rest;
    const res = await sendRequired(
      this.client,
      "/device/retrieve",
      rest,
      retrieveArgsZ,
      retrieveResZ(schemas),
    );
    checkForMultipleOrNoResults("Device", rest, res.devices, isSingle);
    return isSingle ? res.devices[0] : res.devices;
  }

  async create(device: New): Promise<Device>;

  async create(devices: New[]): Promise<Device[]>;

  async create<
    Properties extends z.ZodType,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    device: New<Properties, Make, Model>,
    schemas: DeviceSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async create(device: New): Promise<Device>;

  async create<
    Properties extends z.ZodType,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    devices: New<Properties, Make, Model>[],
    schemas: DeviceSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>[]>;

  async create(devices: New[]): Promise<Device[]>;

  async create(
    devices: New | New[],
    schemas?: DeviceSchemas,
  ): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired(
      this.client,
      "/device/create",
      { devices: array.toArray(devices) },
      createReqZ(schemas),
      createResZ(schemas),
    );
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/device/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
