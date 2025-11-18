// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Payload,
  rackZ,
  type Status,
} from "@/hardware/rack/payload";
import { type task } from "@/hardware/task";
import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_rack_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  embedded: z.boolean().optional(),
  hostIsNode: z.boolean().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const retrieveResZ = z.object({ racks: array.nullableZ(rackZ) });

const singleRetrieveArgsZ = z.union([
  z
    .object({
      key: keyZ,
      includeStatus: z.boolean().optional(),
    })
    .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus })),
  z
    .object({
      name: z.string(),
      includeStatus: z.boolean().optional(),
    })
    .transform(({ name, includeStatus }) => ({ names: [name], includeStatus })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;

const multiRetrieveArgsZ = retrieveReqZ;

export type RetrieveMultipleParams = z.input<typeof multiRetrieveArgsZ>;

const retrieveArgsZ = z.union([singleRetrieveArgsZ, multiRetrieveArgsZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: rackZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;

  constructor(client: UnaryClient, taskClient: task.Client) {
    this.client = client;
    this.tasks = taskClient;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      "/hardware/rack/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      "/hardware/rack/create",
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.racks);
    return isSingle ? sugared[0] : sugared;
  }

  async retrieve(args: RetrieveSingleParams): Promise<Rack>;
  async retrieve(args: RetrieveMultipleParams): Promise<Rack[]>;
  async retrieve(args: RetrieveArgs): Promise<Rack | Rack[]> {
    const isSingle = "key" in args || "name" in args;
    const res = await sendRequired(
      this.client,
      "/hardware/rack/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    const sugared = this.sugar(res.racks);
    checkForMultipleOrNoResults("Rack", args, sugared, isSingle);
    return isSingle ? sugared[0] : sugared;
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(({ key, name, status }) => new Rack(key, name, this.tasks, status));
    return isSingle ? sugared[0] : sugared;
  }
}

export class Rack {
  key: Key;
  name: string;
  status?: Status;
  private readonly tasks: task.Client;

  constructor(key: Key, name: string, taskClient: task.Client, status?: Status) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.status = status;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve({ rack: this.key });
  }

  async createTask(task: task.New): Promise<task.Task>;
  async createTask<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    task: task.New<Type, Config>,
    schemas: task.Schemas<Type, Config, StatusData>,
  ): Promise<task.Task<Type, Config, StatusData>>;

  async createTask<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    task: task.New<Type, Config>,
    schemas?: task.Schemas<Type, Config, StatusData>,
  ): Promise<task.Task<Type, Config, StatusData>> {
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
    return await this.tasks.create(
      task,
      schemas as Required<task.Schemas<Type, Config, StatusData>>,
    );
  }

  async deleteTask(task: task.Key): Promise<void> {
    await this.tasks.delete([task]);
  }

  get payload(): Payload {
    return { key: this.key, name: this.name, status: this.status };
  }
}

export const ontologyID = (key: Key): ontology.ID => ({
  type: "rack",
  key: key.toString(),
});

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
