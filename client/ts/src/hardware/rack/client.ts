// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x/array";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod/v4";

import { framer } from "@/framer";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Payload,
  rackZ,
  type Status,
  statusZ,
} from "@/hardware/rack/payload";
import { type task } from "@/hardware/task";
import { ontology } from "@/ontology";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/hardware/rack/retrieve";
const CREATE_ENDPOINT = "/hardware/rack/create";
const DELETE_ENDPOINT = "/hardware/rack/delete";

export const STATUS_CHANNEL_NAME = "sy_rack_status";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
  embedded: z.boolean().optional(),
  hostIsNode: z.boolean().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  includeStatus: z.boolean().optional(),
});

type RetrieveRequest = z.infer<typeof retrieveReqZ>;

const retrieveResZ = z.object({ racks: nullableArrayZ(rackZ) });

const createReqZ = z.object({ racks: newZ.array() });

const createResZ = z.object({ racks: rackZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

export interface RetrieveOptions extends Pick<RetrieveRequest, "includeStatus"> {}

export class Client implements AsyncTermSearcher<string, Key, Payload> {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;
  private readonly frameClient: framer.Client;

  constructor(
    client: UnaryClient,
    taskClient: task.Client,
    frameClient: framer.Client,
  ) {
    this.client = client;
    this.tasks = taskClient;
    this.frameClient = frameClient;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
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
      CREATE_ENDPOINT,
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.racks);
    if (isSingle) return sugared[0];
    return sugared;
  }

  async search(term: string): Promise<Rack[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { search: term },
      retrieveReqZ,
      retrieveResZ,
    );
    return this.sugar(res.racks);
  }

  async page(offset: number, limit: number): Promise<Rack[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveReqZ,
      retrieveResZ,
    );
    return this.sugar(res.racks);
  }

  async retrieve(key: string | Key, options?: RetrieveOptions): Promise<Rack>;
  async retrieve(keys: Key[], options?: RetrieveOptions): Promise<Rack[]>;
  async retrieve(
    racks: string | Key | Key[],
    options?: RetrieveOptions,
  ): Promise<Rack | Rack[]> {
    const { variant, normalized, single } = analyzeParams(racks, {
      string: "names",
      number: "keys",
    });
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      {
        [variant]: normalized,
        includeStatus: options?.includeStatus,
      },
      retrieveReqZ,
      retrieveResZ,
    );
    const sugared = this.sugar(res.racks);
    checkForMultipleOrNoResults("Rack", racks, sugared, single);
    return single ? sugared[0] : sugared;
  }

  async openStatusObserver(): Promise<framer.ObservableStreamer<Status[]>> {
    return new framer.ObservableStreamer<Status[]>(
      await this.frameClient.openStreamer(STATUS_CHANNEL_NAME),
      (fr) => {
        const data = fr.get(STATUS_CHANNEL_NAME);
        return [data.parseJSON(statusZ), data.length > 0];
      },
    );
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(({ key, name, status }) => new Rack(key, name, this.tasks, status));
    if (isSingle) return sugared[0];
    return sugared;
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

  async deleteTask(task: bigint): Promise<void> {
    await this.tasks.delete([task]);
  }

  get payload(): Payload {
    return { key: this.key, name: this.name, status: this.status };
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
