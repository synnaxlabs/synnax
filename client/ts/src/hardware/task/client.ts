// Copyright 2025 Synnax Labs, Inc.
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
import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import {
  type Command,
  type CommandObservable,
  commandZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Payload,
  type State,
  type StateObservable,
  stateZ,
  taskZ,
} from "@/hardware/task/payload";
import { ontology } from "@/ontology";
import { type ranger } from "@/ranger";
import { signals } from "@/signals";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const STATE_CHANNEL_NAME = "sy_task_state";
const COMMAND_CHANNEL_NAME = "sy_task_cmd";

const NOT_CREATED_ERROR = new Error("Task not created");

export class Task<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  readonly key: Key;
  name: string;
  readonly internal: boolean;
  readonly type: Type;
  config: Config;
  readonly snapshot: boolean;
  state?: State<Details>;
  private readonly frameClient: framer.Client | null;
  private readonly ontologyClient: ontology.Client | null;
  private readonly rangeClient: ranger.Client | null;

  constructor(
    key: Key,
    name: string,
    type: Type,
    config: Config,
    internal: boolean = false,
    snapshot: boolean = false,
    state?: State<Details> | null,
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

  get payload(): Payload<Config, Details> {
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
    return ontologyID(this.key);
  }

  async executeCommand(type: string, args?: UnknownRecord): Promise<string> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    const writer = await this.frameClient.openWriter(COMMAND_CHANNEL_NAME);
    const key = id.id();
    await writer.write(COMMAND_CHANNEL_NAME, [{ task: this.key, type, key, args }]);
    await writer.close();
    return key;
  }

  async executeCommandSync<Details extends {} = UnknownRecord>(
    type: string,
    args: UnknownRecord,
    timeout: CrudeTimeSpan,
  ): Promise<State<Details>> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    const streamer = await this.frameClient.openStreamer(STATE_CHANNEL_NAME);
    const cmdKey = await this.executeCommand(type, args);
    let res: State<Details>;
    const to = new Promise((resolve) =>
      setTimeout(() => resolve(false), new TimeSpan(timeout).milliseconds),
    );
    while (true) {
      const frame = (await Promise.any([streamer.read(), to])) as framer.Frame | false;
      if (frame === false) throw new Error("Command timed out");
      const parsed = stateZ.safeParse(frame.at(-1).sy_task_state);
      if (parsed.success) {
        res = parsed.data as State<Details>;
        if (res.key === cmdKey) break;
      } else throw parsed.error;
    }
    streamer.close();
    return res;
  }

  async openStateObserver(): Promise<StateObservable> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    return new framer.ObservableStreamer<State>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = stateZ.safeParse(s.at(-1));
        if (!parse.success) {
          console.error(parse.error);
          return [null, false];
        }
        const state = parse.data;
        if (state.task !== this.key) return [null, false];
        return [state, true];
      },
    );
  }

  async openCommandObserver(): Promise<CommandObservable> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    return new framer.ObservableStreamer<Command>(
      await this.frameClient.openStreamer(COMMAND_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(COMMAND_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = commandZ.safeParse(s.at(-1));
        if (!parse.success) {
          console.error(parse.error);
          return [null, false];
        }
        const cmd = parse.data;
        if (cmd.task !== this.key) return [null, false];
        return [cmd, true];
      },
    );
  }

  async snapshottedTo(): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null || this.rangeClient == null)
      throw NOT_CREATED_ERROR;
    if (!this.snapshot) return null;
    const parents = await this.ontologyClient.retrieveParents(this.ontologyID);
    if (parents.length == 0) return null;
    return parents[0];
  }
}

const retrieveReqZ = z.object({
  rack: rackKeyZ.optional(),
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
  includeState: z.boolean().optional(),
});

const retrieveResZ = z.object({ tasks: nullableArrayZ(taskZ) });

export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

export interface RetrieveOptions
  extends Pick<RetrieveRequest, "rack" | "offset" | "limit" | "includeState"> {}

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";
const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";
const COPY_ENDPOINT = "/hardware/task/copy";

const createReqZ = z.object({ tasks: newZ.array() });
const createResZ = z.object({ tasks: taskZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });
const copyResZ = z.object({ task: taskZ });

export class Client implements AsyncTermSearcher<string, Key, Payload> {
  readonly type: string = ONTOLOGY_TYPE;
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
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(task: New<Config, Type>): Promise<Task<Config, Details, Type>>;

  async create<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(tasks: New<Config, Type>[]): Promise<Task<Config, Details, Type>[]>;

  async create<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(
    task: New<Config, Type> | Array<New<Config, Type>>,
  ): Promise<Task<Config, Details, Type> | Array<Task<Config, Details, Type>>> {
    const isSingle = !Array.isArray(task);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { tasks: toArray(task) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.tasks) as Array<Task<Config, Details, Type>>;
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
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(rack: number, options?: RetrieveOptions): Promise<Task<Config, Details, Type>[]>;

  async retrieve<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(keys: string[], options?: RetrieveOptions): Promise<Task<Config, Details, Type>[]>;

  async retrieve<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(key: string, options?: RetrieveOptions): Promise<Task<Config, Details, Type>>;

  async retrieve<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(
    rack: number | string | string[],
    options?: RetrieveOptions,
  ): Promise<Task<Config, Details, Type> | Task<Config, Details, Type>[]> {
    const { single, normalized, variant } = analyzeParams(
      rack,
      { number: "rack", string: "keys" },
      { convertNumericStrings: false },
    );
    const req: RetrieveRequest = { ...options };
    if (variant === "rack") req.rack = rack as number;
    else req.keys = normalized as string[];
    const tasks = await this.execRetrieve(req);
    const sugared = this.sugar(tasks) as Array<Task<Config, Details, Type>>;
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

  async retrieveByName<
    Config extends UnknownRecord = UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(name: string, rack?: number): Promise<Task<Config, Details, Type>> {
    const tasks = await this.execRetrieve({ names: [name], rack });
    checkForMultipleOrNoResults("Task", name, tasks, true);
    return this.sugar(tasks)[0] as Task<Config, Details, Type>;
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

  async openStateObserver(): Promise<StateObservable> {
    return new framer.ObservableStreamer<State>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = stateZ.safeParse(s.at(-1));
        if (!parse.success) {
          console.error(parse.error);
          return [null, false];
        }
        return [parse.data, true];
      },
    );
  }

  async openCommandObserver(): Promise<CommandObservable> {
    return new framer.ObservableStreamer<Command>(
      await this.frameClient.openStreamer(COMMAND_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(COMMAND_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = commandZ.safeParse(s.at(-1));
        if (!parse.success) {
          console.error(parse.error);
          return [null, false];
        }
        return [parse.data, true];
      },
    );
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
