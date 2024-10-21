// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { id } from "@synnaxlabs/x";
import { type UnknownRecord } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { framer } from "@/framer";
import { type Frame } from "@/framer/frame";
import { rack } from "@/hardware/rack";
import {
  NewTask,
  newTaskZ,
  Payload,
  State,
  StateObservable,
  stateZ,
  TaskKey,
  taskKeyZ,
  taskZ,
} from "@/hardware/task/payload";
import { ontology } from "@/ontology";
import { ranger } from "@/ranger";
import { signals } from "@/signals";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const TASK_STATE_CHANNEL = "sy_task_state";
const TASK_CMD_CHANNEL = "sy_task_cmd";

const TASK_NOT_CREATED = new Error("Task not created");

export class Task<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  readonly key: TaskKey;
  readonly name: string;
  readonly internal: boolean;
  readonly type: T;
  config: C;
  readonly snapshot: boolean;
  state?: State<D>;
  private readonly frameClient: framer.Client | null;
  private readonly ontologyClient: ontology.Client | null;
  private readonly rangeClient: ranger.Client | null;

  constructor(
    key: TaskKey,
    name: string,
    type: T,
    config: C,
    internal: boolean = false,
    snapshot: boolean = false,
    state?: State<D> | null,
    frameClient: framer.Client | null = null,
    ontologyClient: ontology.Client | null = null,
    rangeClient: ranger.Client | null = null,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.internal = internal;
    this.snapshot = snapshot;
    if (state !== null) this.state = state;
    this.frameClient = frameClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
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

  get ontologyID(): ontology.ID {
    return new ontology.ID({ type: "task", key: this.key });
  }

  async executeCommand(type: string, args?: UnknownRecord): Promise<string> {
    if (this.frameClient == null) throw TASK_NOT_CREATED;
    const writer = await this.frameClient.openWriter(TASK_CMD_CHANNEL);
    const key = id.id();
    await writer.write(TASK_CMD_CHANNEL, [{ task: this.key, type, key, args }]);
    await writer.close();
    return key;
  }

  async executeCommandSync<D extends UnknownRecord = UnknownRecord>(
    type: string,
    args: UnknownRecord,
    timeout: CrudeTimeSpan,
  ): Promise<State<D>> {
    if (this.frameClient == null) throw TASK_NOT_CREATED;
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
    if (this.frameClient == null) throw TASK_NOT_CREATED;
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

  async snapshottedTo(): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null || this.rangeClient == null) throw TASK_NOT_CREATED;
    if (!this.snapshot) return null;
    const parents = await this.ontologyClient.retrieveParents(this.ontologyID);
    if (parents.length == 0) return null;
    return parents[0];
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
const COPY_ENDPOINT = "/hardware/task/copy";

const createReqZ = z.object({ tasks: newTaskZ.array() });
const createResZ = z.object({ tasks: taskZ.array() });
const deleteReqZ = z.object({ keys: taskKeyZ.array() });
const deleteResZ = z.object({});
const copyReqZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  snapshot: z.boolean(),
});
const copyResZ = z.object({ task: taskZ });

export class Client implements AsyncTermSearcher<string, TaskKey, Payload> {
  readonly type: string = "task";
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: ranger.Client;

  constructor(
    client: UnaryClient,
    frameClient: framer.Client,
    ontologyClient: ontology.Client,
    rangeClient: ranger.Client,
  ) {
    this.client = client;
    this.frameClient = frameClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
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
    return await this.execRetrieve({ offset, limit });
  }

  async list(options: RetrieveOptions = {}): Promise<Task[]> {
    return this.sugar(await this.execRetrieve(options));
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

  async copy(key: string, name: string, snapshot: boolean): Promise<Task> {
    const res = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyResZ,
    );
    return this.sugar([res.task])[0];
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
      ({ key, name, type, config, state, internal, snapshot }) =>
        new Task(
          key,
          name,
          type,
          config,
          internal,
          snapshot,
          state,
          this.frameClient,
          this.ontologyClient,
          this.rangeClient,
        ),
    );
  }

  async openTracker(): Promise<signals.Observable<string, string>> {
    return await signals.openObservable<string, string>(
      this.frameClient,
      "sy_task_set",
      "sy_task_delete",
      (variant, data) =>
        Array.from(data).map((k) => ({
          variant,
          key: k.toString(),
          value: k.toString(),
        })),
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
