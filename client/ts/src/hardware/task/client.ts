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
import { array } from "@synnaxlabs/x/array";
import { type record } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x/telem";
import { z } from "zod/v4";

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
  type StateObservable,
  type Status,
  statusZ,
  taskZ,
} from "@/hardware/task/payload";
import { ontology } from "@/ontology";
import { type ranger } from "@/ranger";
import { signals } from "@/signals";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const STATE_CHANNEL_NAME = "sy_task_state";
const COMMAND_CHANNEL_NAME = "sy_task_cmd";
const SET_CHANNEL_NAME = "sy_task_set";
const DELETE_CHANNEL_NAME = "sy_task_delete";

const NOT_CREATED_ERROR = new Error("Task not created");

const retrieveSnapshottedTo = async (taskKey: Key, ontologyClient: ontology.Client) => {
  const task = await ontologyClient.retrieveParents(taskKey);
  if (task.length === 0) return null;
  return task[0];
};

export class Task<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodTypeAny = z.ZodTypeAny,
> {
  readonly key: Key;
  name: string;
  readonly internal: boolean;
  readonly type: z.infer<Type>;
  config: z.infer<Config>;
  readonly snapshot: boolean;
  status?: Status<StatusData>;
  private readonly frameClient: framer.Client | null;
  private readonly ontologyClient: ontology.Client | null;
  private readonly rangeClient: ranger.Client | null;
  private readonly stateSchema: z.ZodType<StatusData>;

  constructor(
    key: Key,
    name: string,
    type: z.infer<Type>,
    config: z.infer<Config>,
    internal: boolean = false,
    snapshot: boolean = false,
    status?: Status<StatusData> | null,
    frameClient: framer.Client | null = null,
    ontologyClient: ontology.Client | null = null,
    rangeClient: ranger.Client | null = null,
    stateSchema: z.ZodType<StatusData> = z.unknown() as unknown as z.ZodType<StatusData>,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.internal = internal;
    this.snapshot = snapshot;
    if (status !== null) this.status = status;
    this.frameClient = frameClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
    this.stateSchema = stateSchema;
  }

  get payload(): Payload<Type, Config, StatusData> {
    return {
      key: this.key,
      name: this.name,
      type: this.type,
      config: this.config,
      status: this.status,
      internal: this.internal,
    };
  }

  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }

  async executeCommand<Args>(type: string, args?: Args): Promise<string> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    const writer = await this.frameClient.openWriter(COMMAND_CHANNEL_NAME);
    const key = id.create();
    await writer.write(COMMAND_CHANNEL_NAME, [{ task: this.key, type, key, args }]);
    await writer.close();
    return key;
  }

  async executeCommandSync<StatusData extends z.ZodTypeAny = z.ZodTypeAny>(
    type: string,
    args: record.Unknown,
    timeout: CrudeTimeSpan,
  ): Promise<Status<StatusData>> {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    const streamer = await this.frameClient.openStreamer(STATE_CHANNEL_NAME);
    const cmdKey = await this.executeCommand(type, args);
    let res: Status<StatusData>;
    const to = new Promise((resolve) =>
      setTimeout(() => resolve(false), new TimeSpan(timeout).milliseconds),
    );
    while (true) {
      const frame = (await Promise.any([streamer.read(), to])) as framer.Frame | false;
      if (frame === false) throw new Error("Command timed out");
      res = statusZ(this.stateSchema).parse(frame.at(-1).sy_task_state);
      if (res.key === cmdKey) break;
    }
    streamer.close();
    return res;
  }

  async openStateObserver<StatusData extends z.ZodTypeAny = z.ZodTypeAny>(): Promise<
    StateObservable<StatusData>
  > {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    return new framer.ObservableStreamer<Status<StatusData>>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = statusZ(this.stateSchema).safeParse(s.at(-1));
        if (!parse.success) {
          console.error(parse.error);
          return [null, false];
        }
        const status = parse.data;
        if (status.details.task !== this.key) return [null, false];
        return [status, true];
      },
    );
  }

  async openCommandObserver<Args extends z.ZodTypeAny = z.ZodTypeAny>(): Promise<
    CommandObservable<Args>
  > {
    if (this.frameClient == null) throw NOT_CREATED_ERROR;
    return new framer.ObservableStreamer<Command<Args>>(
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
        return [cmd as Command<Args>, true];
      },
    );
  }

  async snapshottedTo(): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null || this.rangeClient == null)
      throw NOT_CREATED_ERROR;
    if (!this.snapshot) return null;
    return await retrieveSnapshottedTo(this.key, this.ontologyClient);
  }
}

const retrieveReqZ = z.object({
  rack: rackKeyZ.optional(),
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  types: z.string().array().optional(),
  includeStatus: z.boolean().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const retrieveResZ = z.object({ tasks: nullableArrayZ(taskZ()) });

export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

export interface RetrieveOptions
  extends Pick<
    RetrieveRequest,
    "rack" | "offset" | "limit" | "includeStatus" | "types"
  > {}

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";
const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";
const COPY_ENDPOINT = "/hardware/task/copy";

const createReqZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodTypeAny = z.ZodTypeAny,
>(
  typeZ: Type = z.string() as unknown as Type,
  configZ: Config = z.unknown() as unknown as Config,
  statusDataZ: StatusData = z.unknown() as unknown as StatusData,
) => z.object({ tasks: newZ(typeZ, configZ, statusDataZ).array() });
const createResZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodTypeAny = z.ZodTypeAny,
>(
  typeZ: Type = z.string() as unknown as Type,
  configZ: Config = z.unknown() as unknown as Config,
  statusDataZ: StatusData = z.unknown() as unknown as StatusData,
) => z.object({ tasks: taskZ(typeZ, configZ, statusDataZ).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });
const copyResZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodTypeAny = z.ZodTypeAny,
>(
  typeZ: Type = z.string() as unknown as Type,
  configZ: Config = z.unknown() as unknown as Config,
  statusDataZ: StatusData = z.unknown() as unknown as StatusData,
) => z.object({ task: taskZ(typeZ, configZ, statusDataZ) });

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
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(task: New<Type, Config>): Promise<Task<Type, Config, StatusData>>;
  async create<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(tasks: New<Type, Config>[]): Promise<Task<Type, Config, StatusData>[]>;

  async create<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    task: New<Type, Config> | Array<New<Type, Config>>,
    typeZ: Type = z.string() as unknown as Type,
    configZ: Config = z.unknown() as unknown as Config,
    statusDataZ: StatusData = z.unknown() as unknown as StatusData,
  ): Promise<Task<Type, Config, StatusData> | Array<Task<Type, Config, StatusData>>> {
    const isSingle = !Array.isArray(task);
    const createReq = createReqZ(typeZ, configZ, statusDataZ);
    const createRes = createResZ(typeZ, configZ, statusDataZ);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { tasks: array.toArray(task) } as z.infer<typeof createReq>,
      createReq,
      createRes,
    );
    const sugared = this.sugar<Type, Config, StatusData>(
      res.tasks as Payload<Type, Config, StatusData>[],
    );
    return isSingle ? sugared[0] : sugared;
  }

  async delete(keys: bigint | bigint[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
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
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(rack: number, options?: RetrieveOptions): Promise<Task<Type, Config, StatusData>[]>;

  async retrieve<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    keys: string[],
    options?: RetrieveOptions,
  ): Promise<Task<Type, Config, StatusData>[]>;

  async retrieve<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(key: string, options?: RetrieveOptions): Promise<Task<Type, Config, StatusData>>;

  async retrieve<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    rack: number | string | string[],
    options?: RetrieveOptions,
  ): Promise<Task<Type, Config, StatusData> | Task<Type, Config, StatusData>[]> {
    const { single, normalized, variant } = analyzeParams(
      rack,
      { number: "rack", string: "keys" },
      { convertNumericStrings: false },
    );
    const req: RetrieveRequest = { ...options };
    if (variant === "rack") req.rack = rack as number;
    else req.keys = normalized as string[];
    const tasks = await this.execRetrieve(req);
    const sugared = this.sugar<Type, Config, StatusData>(
      tasks as Payload<Type, Config, StatusData>[],
    );
    return single && variant !== "rack" ? sugared[0] : sugared;
  }

  async copy<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    key: string,
    name: string,
    snapshot: boolean,
    typeZ: Type = z.string() as unknown as Type,
    configZ: Config = z.unknown() as unknown as Config,
    statusDataZ: StatusData = z.unknown() as unknown as StatusData,
  ): Promise<Task<Type, Config, StatusData>> {
    const copyRes = copyResZ(typeZ, configZ, statusDataZ);
    const response = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyRes,
    );
    return this.sugar<Type, Config, StatusData>(
      response.task as Payload<Type, Config, StatusData>,
    );
  }

  async retrieveByName<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(name: string, rack?: number): Promise<Task<Type, Config, StatusData>> {
    const tasks = await this.execRetrieve({ names: [name], rack });
    checkForMultipleOrNoResults("Task", name, tasks, true);
    return this.sugar(tasks)[0] as Task<Type, Config, StatusData>;
  }

  async retrieveByType<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(type: z.infer<Type>, rack?: number): Promise<Task<Type, Config, StatusData>[]> {
    const tasks = await this.execRetrieve({ types: [type], rack });
    return this.sugar(tasks) as Task<Type, Config, StatusData>[];
  }

  async retrieveSnapshottedTo(taskKey: Key): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null) throw NOT_CREATED_ERROR;
    return await retrieveSnapshottedTo(taskKey, this.ontologyClient);
  }

  private async execRetrieve<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(req: RetrieveRequest): Promise<Payload<Type, Config, StatusData>[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.tasks as Payload<Type, Config, StatusData>[];
  }

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(payload: Payload<Type, Config, StatusData>): Task<Type, Config, StatusData>;

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(payloads: Payload<Type, Config, StatusData>[]): Task<Type, Config, StatusData>[];

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodTypeAny = z.ZodTypeAny,
  >(
    payloads: Payload<Type, Config, StatusData> | Payload<Type, Config, StatusData>[],
  ): Task<Type, Config, StatusData>[] | Task<Type, Config, StatusData> {
    const isSingle = !Array.isArray(payloads);
    const res = array
      .toArray(payloads)
      .map(
        ({ key, name, type, config, status: state, internal, snapshot }) =>
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
    return isSingle ? res[0] : res;
  }

  async openTracker(): Promise<signals.Observable<string, string>> {
    return await signals.openObservable<string, string>(
      this.frameClient,
      SET_CHANNEL_NAME,
      DELETE_CHANNEL_NAME,
      (variant, data) =>
        Array.from(data).map((k) => ({
          variant,
          key: k.toString(),
          value: k.toString(),
        })),
    );
  }

  async openStateObserver<StatusData extends z.ZodTypeAny = z.ZodTypeAny>(
    stateSchema: z.ZodType<StatusData> = z.unknown() as unknown as z.ZodType<StatusData>,
  ): Promise<StateObservable<StatusData>> {
    return new framer.ObservableStreamer<Status<StatusData>>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const parse = statusZ(stateSchema).safeParse(s.at(-1));
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
        const parse = commandZ.safeParse(s.at(-1) as unknown as Command);
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
