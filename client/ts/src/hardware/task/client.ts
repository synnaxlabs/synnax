// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { binary, type observe } from "@synnaxlabs/x";
import { type UnknownRecord } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { nanoid } from "nanoid/non-secure";
import { z } from "zod";

import { framer } from "@/framer";
import { type Frame } from "@/framer/frame";
import { rack } from "@/hardware/rack";
import { signals } from "@/signals";
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

export const stateZ = z.object({
  task: taskKeyZ,
  variant: z.string(),
  key: z.string().optional(),
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

export type State<D extends {} = UnknownRecord> = Omit<
  z.infer<typeof stateZ>,
  "details"
> & {
  details?: D;
};

export const taskZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  type: z.string(),
  internal: z.boolean().optional(),
  config: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return binary.JSON_ECD.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
  state: stateZ.optional().nullable(),
});

export const newTaskZ = taskZ.omit({ key: true }).extend({
  key: taskKeyZ.transform((k) => k.toString()).optional(),
  config: z.unknown().transform((c) => binary.JSON_ECD.encodeString(c)),
});

export type NewTask<
  C extends UnknownRecord = UnknownRecord,
  T extends string = string,
> = Omit<z.input<typeof newTaskZ>, "config" | "state"> & {
  type: T;
  config: C;
};

export type Payload<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> = Omit<z.output<typeof taskZ>, "config" | "type" | "state"> & {
  type: T;
  config: C;
  state?: State<D> | null;
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

export type StateObservable<D extends UnknownRecord = UnknownRecord> =
  observe.ObservableAsyncCloseable<State<D>>;

export class Task<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  readonly key: TaskKey;
  readonly name: string;
  readonly internal: boolean;
  readonly type: T;
  readonly config: C;
  state?: State<D>;
  private readonly frameClient: framer.Client;

  constructor(
    key: TaskKey,
    name: string,
    type: T,
    config: C,
    frameClient: framer.Client,
    internal: boolean = false,
    state?: State<D> | null,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.internal = internal;
    if (state !== null) this.state = state;
    this.frameClient = frameClient;
  }

  get payload(): Payload<C, D> {
    return {
      key: this.key,
      name: this.name,
      type: this.type,
      config: this.config,
      state: this.state,
      internal: this.internal,
    };
  }

  async executeCommand(type: string, args?: UnknownRecord): Promise<string> {
    const writer = await this.frameClient.openWriter(TASK_CMD_CHANNEL);
    const key = nanoid();
    await writer.write(TASK_CMD_CHANNEL, [{ task: this.key, type, key, args }]);
    await writer.close();
    return key;
  }

  async executeCommandSync<D extends UnknownRecord = UnknownRecord>(
    type: string,
    args: UnknownRecord,
    timeout: CrudeTimeSpan,
  ): Promise<State<D>> {
    const streamer = await this.frameClient.openStreamer(TASK_STATE_CHANNEL);
    const cmdKey = await this.executeCommand(type, args);
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
        if (res.key === cmdKey) break;
      } else {
        console.error(parsed.error);
      }
    }
    streamer.close();
    return res;
  }

  async openStateObserver<D extends UnknownRecord = UnknownRecord>(): Promise<
    StateObservable<D>
  > {
    return new framer.ObservableStreamer<State<D>>(
      await this.frameClient.openStreamer(TASK_STATE_CHANNEL),
      (frame) => {
        const s = frame.get(TASK_STATE_CHANNEL);
        if (s.length === 0) return [null, false];
        const parse = stateZ.safeParse(s.at(-1));
        if (!parse.success) return [null, false];
        const state = parse.data as State<D>;
        if (state.task !== this.key) return [null, false];
        return [state, true];
      },
    );
  }
}

const retrieveReqZ = z.object({
  rack: rack.rackKeyZ.optional(),
  keys: z.string().array().optional(),
  names: z.string().array().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
  includeState: z.boolean().optional(),
});

const retrieveResZ = z.object({
  tasks: nullableArrayZ(taskZ),
});

export type RetrieveRequest = z.infer<typeof retrieveReqZ>;

export type RetrieveOptions = Pick<
  RetrieveRequest,
  "rack" | "offset" | "limit" | "includeState"
>;

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";
const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";

const createReqZ = z.object({ tasks: newTaskZ.array() });
const createResZ = z.object({ tasks: taskZ.array() });
const deleteReqZ = z.object({ keys: taskKeyZ.array() });
const deleteResZ = z.object({});

export class Client implements AsyncTermSearcher<string, TaskKey, Payload> {
  readonly type: string = "task";
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async create<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(task: NewTask<C, T>): Promise<Task<C, D, T>>;

  async create<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(tasks: NewTask<C, T>[]): Promise<Task<C, D, T>[]>;

  async create<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(
    task: NewTask<C, T> | Array<NewTask<C, T>>,
  ): Promise<Task<C, D, T> | Array<Task<C, D, T>>> {
    const isSingle = !Array.isArray(task);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { tasks: toArray(task) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.tasks) as Array<Task<C, D, T>>;
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

  async search(term: string): Promise<Payload[]> {
    return await this.execRetrieve({ keys: [term] });
  }

  async page(offset: number, limit: number): Promise<Payload[]> {
    return this.execRetrieve({ offset, limit });
  }

  async list(options: RetrieveOptions = {}): Promise<Payload[]> {
    return this.execRetrieve(options);
  }

  async retrieve<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(rack: number, options?: RetrieveOptions): Promise<Task<C, D, T>[]>;

  async retrieve<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(keys: string[], options?: RetrieveOptions): Promise<Task<C, D, T>[]>;

  async retrieve<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(key: string, options?: RetrieveOptions): Promise<Task<C, D, T>>;

  async retrieve<
    C extends UnknownRecord = UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(
    rack: number | string | string[],
    options?: RetrieveOptions,
  ): Promise<Task<C, D, T> | Task<C, D, T>[]> {
    const { single, normalized, variant } = analyzeParams(
      rack,
      { number: "rack", string: "keys" },
      { convertNumericStrings: false },
    );
    const req: RetrieveRequest = { ...options };
    if (variant === "rack") req.rack = rack as number;
    else req.keys = normalized as string[];
    const tasks = await this.execRetrieve(req);
    const sugared = this.sugar(tasks) as Array<Task<C, D, T>>;
    return single && variant !== "rack" ? sugared[0] : sugared;
  }

  async retrieveByName(name: string, rack?: number): Promise<Task> {
    const tasks = await this.execRetrieve({ names: [name], rack });
    checkForMultipleOrNoResults("Task", name, tasks, true);
    return this.sugar(tasks)[0];
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Payload[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.tasks;
  }

  private sugar(payloads: Payload[]): Task[] {
    return payloads.map(
      ({ key, name, type, config, state, internal }) =>
        new Task(key, name, type, config, this.frameClient, internal, state),
    );
  }

  async openTracker(): Promise<signals.Observable<string, Task>> {
    return await signals.openObservable<string, Task>(
      this.frameClient,
      "sy_task_set",
      "sy_task_delete",
      (variant, data) => {
        if (variant === "delete") {
          return Array.from(data).map((k) => ({
            variant,
            key: k.toString(),
            value: undefined,
          }));
        }
        const sugared = this.sugar(data.parseJSON(taskZ));
        return sugared.map((t) => ({ variant, key: t.key, value: t }));
      },
    );
  }

  async openStateObserver<D extends UnknownRecord = UnknownRecord>(): Promise<
    StateObservable<D>
  > {
    return new framer.ObservableStreamer<State<D>>(
      await this.frameClient.openStreamer(TASK_STATE_CHANNEL),
      (frame) => {
        const s = frame.get(TASK_STATE_CHANNEL);
        if (s.length === 0) return [null, false];
        const parse = stateZ.safeParse(s.at(-1));
        if (!parse.success) return [null, false];
        return [parse.data as State<D>, true];
      },
    );
  }
}
