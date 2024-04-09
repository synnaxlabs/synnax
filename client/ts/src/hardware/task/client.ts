// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { rack } from "@/hardware/rack";
import { nullableArrayZ } from "@/util/zod";

export const taskKeyZ = z.union([
  z.string(),
  z.bigint().transform((k) => k.toString()),
  z.number().transform((k) => k.toString()),
]);

export type TaskKey = z.infer<typeof taskKeyZ>;

export const taskZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  type: z.string(),
  config: z
    .record(z.unknown())
    .or(z.string().transform((c) => JSON.parse(c))) as z.ZodType<UnknownRecord>,
});

export const newTaskZ = taskZ.omit({ key: true }).extend({
  key: taskKeyZ.transform((k) => k.toString()).optional(),
  config: z.unknown().transform((c) => JSON.stringify(c)),
});

export type NewTask = z.input<typeof newTaskZ>;

export type Task<
  T extends string = string,
  C extends UnknownRecord = UnknownRecord,
> = Omit<z.infer<typeof taskZ>, "config" | "type"> & { type: T; config: C };

const retrieveReqZ = z.object({
  rack: rack.rackKeyZ.optional(),
  keys: z.string().array().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const retrieveResZ = z.object({
  tasks: nullableArrayZ(taskZ),
});

export type RetrieveRequest = z.infer<typeof retrieveReqZ>;

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";

const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";

const createReqZ = z.object({
  tasks: newTaskZ.array(),
});

const createResZ = z.object({
  tasks: taskZ.array(),
});

const deleteReqZ = z.object({
  keys: taskKeyZ.array(),
});

const deleteResZ = z.object({});

export class Client implements AsyncTermSearcher<string, TaskKey, Task> {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(task: NewTask): Promise<Task>;

  async create(tasks: NewTask[]): Promise<Task[]>;

  async create(task: NewTask | NewTask[]): Promise<Task | Task[]> {
    const isSingle = !Array.isArray(task);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { tasks: toArray(task) },
      createReqZ,
      createResZ,
    );
    return isSingle ? res.tasks[0] : res.tasks;
  }

  async delete(keys: bigint | bigint[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async search(term: string): Promise<Task[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: [term] },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.tasks;
  }

  async page(offset: number, limit: number): Promise<Task[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.tasks;
  }

  async retrieve(rack: number): Promise<Task[]>;

  async retrieve(keys: string[]): Promise<Task[]>;

  async retrieve(key: string): Promise<Task>;

  async retrieve(rack: number | string | string[]): Promise<Task | Task[]> {
    const params: RetrieveRequest = {};
    let multi: boolean = true;
    if (typeof rack === "number") params.rack = rack;
    else if (typeof rack === "string") {
      multi = false;
      params.keys = [rack];
    } else params.keys = toArray(rack);
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      params,
      retrieveReqZ,
      retrieveResZ,
    );
    return multi ? res.tasks : res.tasks[0];
  }
}
