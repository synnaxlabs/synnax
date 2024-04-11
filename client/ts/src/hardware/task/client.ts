// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { binary, type observe } from "@synnaxlabs/x";
import { type UnknownRecord } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { TimeSpan, type CrudeTimeSpan } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { nanoid } from "nanoid";
import { z } from "zod";

import { type framer } from "@/framer";
import { type Frame } from "@/framer/frame";
import { rack } from "@/hardware/rack";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const TASK_STATE_CHANNEL = "sy_task_state";
const TASK_CMD_CHANNEL = "sy_task_cmd";

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
  config: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return binary.JSON_ECD.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export const newTaskZ = taskZ.omit({ key: true }).extend({
  key: taskKeyZ.transform((k) => k.toString()).optional(),
  config: z.unknown().transform((c) => binary.JSON_ECD.encodeString(c)),
});

export type NewTask = z.input<typeof newTaskZ>;

export type TaskPayload<
  T extends string = string,
  C extends UnknownRecord = UnknownRecord,
> = Omit<z.output<typeof taskZ>, "config" | "type"> & { type: T; config: C };

export const stateZ = z.object({
  task: taskKeyZ,
  variant: z.string(),
  key: z.string(),
  details: z
    .record(z.unknown())
    .or(
      z.string().transform((c) => {
        if (c === "") return {};
        return JSON.parse(c);
      }),
    )
    .or(z.array(z.unknown()))
    .or(z.null()),
});

type State<D extends UnknownRecord = UnknownRecord> = Omit<
  z.infer<typeof stateZ>,
  "details"
> & {
  details: D;
};

export const commandZ = z.object({
  task: taskKeyZ,
  type: z.string(),
  key: z.string(),
  args: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return JSON.parse(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export class Task<T extends string = string, C extends UnknownRecord = UnknownRecord> {
  readonly key: TaskKey;
  readonly name: string;
  readonly type: T;
  readonly config: C;
  private readonly frameClient: framer.Client;

  constructor(
    key: TaskKey,
    name: string,
    type: T,
    config: C,
    frameClient: framer.Client,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.frameClient = frameClient;
  }

  async executeCommandSync<D extends UnknownRecord = UnknownRecord>(
    type: string,
    args: UnknownRecord,
    timeout: CrudeTimeSpan,
  ): Promise<State<D>> {
    const streamer = await this.frameClient.openStreamer(TASK_STATE_CHANNEL);
    const writer = await this.frameClient.openWriter(TASK_CMD_CHANNEL);
    const key = nanoid();
    await writer.write(TASK_STATE_CHANNEL, [{ task: this.key, type, key, args }]);
    await writer.close();
    let res: State<D>;
    const to = new Promise((resolve) =>
      setTimeout(() => resolve(false), new TimeSpan(timeout).milliseconds),
    );
    while (true) {
      const frame = (await Promise.any([streamer.read(), to])) as Frame | false;
      if (frame === false) throw new Error("Command timed out");
      const parsed = stateZ.safeParse(frame.at(-1).sy_task_state);
      if (parsed.success) {
        res = parsed.data as State<D>;
        if (res.key === key) break;
      } else {
        console.error(parsed.error);
      }
    }
    streamer.close();
    return res;
  }

  async observeState<D extends UnknownRecord = UnknownRecord>(): Promise<
    observe.Observable<State<D>>
  > {}
}

const retrieveReqZ = z.object({
  rack: rack.rackKeyZ.optional(),
  keys: z.string().array().optional(),
  names: z.string().array().optional(),
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

export class Client implements AsyncTermSearcher<string, TaskKey, TaskPayload> {
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
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
    const sugared = this.sugar(res.tasks);
    return isSingle ? sugared[0] : sugared;
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

  async search(term: string): Promise<TaskPayload[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: [term] },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.tasks;
  }

  async page(offset: number, limit: number): Promise<TaskPayload[]> {
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
    const { single, normalized, variant } = analyzeParams(
      rack,
      {
        number: "rack",
        string: "keys",
      },
      { convertNumericStrings: false },
    );
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      variant === "rack" ? { rack: rack as number } : { keys: normalized as string[] },
      retrieveReqZ,
      retrieveResZ,
    );
    const sugared = this.sugar(res.tasks);
    return single && variant !== "rack" ? sugared[0] : sugared;
  }

  async retrieveByName(name: string, rack?: number): Promise<Task> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { names: [name], rack },
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Task", name, res.tasks, true);
    return this.sugar(res.tasks)[0];
  }

  private sugar(payloads: TaskPayload[]): Task[] {
    return payloads.map(
      ({ key, name, type, config }) =>
        new Task(key, name, type, config, this.frameClient),
    );
  }
}
