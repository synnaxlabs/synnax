// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import {
  array,
  caseconv,
  type CrudeTimeSpan,
  id,
  type record,
  strings,
  TimeSpan,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type framer } from "@/framer";
import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Payload,
  type Schemas,
  type Status,
  statusZ,
  taskZ,
} from "@/hardware/task/payload";
import { type ontology } from "@/ontology";
import { type ranger } from "@/ranger";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const STATUS_CHANNEL_NAME = "sy_task_status";
export const COMMAND_CHANNEL_NAME = "sy_task_cmd";
export const SET_CHANNEL_NAME = "sy_task_set";
export const DELETE_CHANNEL_NAME = "sy_task_delete";

const NOT_CREATED_ERROR = new Error("Task not created");

const retrieveSnapshottedTo = async (taskKey: Key, ontologyClient: ontology.Client) => {
  const parents = await ontologyClient.retrieveParents(ontologyID(taskKey));
  if (parents.length === 0) return null;
  return parents[0];
};

export interface TaskExecuteCommandParams {
  type: string;
  args?: record.Unknown;
}

export interface ExecuteCommandParams extends TaskExecuteCommandParams {
  task: Key;
}

export interface ExecuteCommandsParams {
  commands: NewCommand[];
}

export interface TaskExecuteCommandSyncParams extends TaskExecuteCommandParams {
  timeout?: CrudeTimeSpan;
}

export interface ExecuteCommandsSyncParams<StatusData extends z.ZodType>
  extends Omit<ExecuteCommandsSyncInternalParams<StatusData>, "frameClient" | "name"> {}

export interface ExecuteCommandSyncParams<StatusData extends z.ZodType>
  extends Omit<ExecuteCommandSyncInternalParams<StatusData>, "frameClient" | "name"> {}

export class Task<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  readonly key: Key;
  name: string;
  internal: boolean;
  type: z.infer<Type>;
  snapshot: boolean;
  config: z.infer<Config>;
  status?: Status<StatusData>;

  readonly schemas: Schemas<Type, Config, StatusData>;
  private readonly frameClient_?: framer.Client;
  private readonly ontologyClient_?: ontology.Client;
  private readonly rangeClient_?: ranger.Client;

  get frameClient(): framer.Client {
    if (this.frameClient_ == null) throw NOT_CREATED_ERROR;
    return this.frameClient_;
  }

  get ontologyClient(): ontology.Client {
    if (this.ontologyClient_ == null) throw NOT_CREATED_ERROR;
    return this.ontologyClient_;
  }

  get rangeClient(): ranger.Client {
    if (this.rangeClient_ == null) throw NOT_CREATED_ERROR;
    return this.rangeClient_;
  }

  constructor(
    {
      key,
      type,
      name,
      config,
      internal = false,
      snapshot = false,
      status,
    }: Payload<Type, Config, StatusData>,
    schemas?: Schemas<Type, Config, StatusData>,
    frameClient?: framer.Client,
    ontologyClient?: ontology.Client,
    rangeClient?: ranger.Client,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.schemas = schemas ?? {
      typeSchema: z.string() as unknown as Type,
      configSchema: z.unknown() as unknown as Config,
      statusDataSchema: z.unknown() as unknown as StatusData,
    };
    this.internal = internal;
    this.snapshot = snapshot;
    this.status = status;
    this.frameClient_ = frameClient;
    this.ontologyClient_ = ontologyClient;
    this.rangeClient_ = rangeClient;
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

  async executeCommand(params: TaskExecuteCommandParams): Promise<string> {
    return await executeCommand({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
    });
  }

  async executeCommandSync(
    params: TaskExecuteCommandSyncParams,
  ): Promise<Status<StatusData>> {
    return await executeCommandSync<StatusData>({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
      name: this.name,
      statusDataZ: this.schemas?.statusDataSchema,
    });
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
  internal: z.boolean().optional(),
  snapshot: z.boolean().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const singleRetrieveArgsZ = z.union([
  z
    .object({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus })),
  z
    .object({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }) => ({ names: [name], includeStatus })),
  z
    .object({
      type: z.string(),
      rack: rackKeyZ.optional(),
    })
    .transform(({ type, rack }) => ({ types: [type], rack })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;

const multiRetrieveArgsZ = retrieveReqZ;
export type RetrieveMultipleParams = z.input<typeof multiRetrieveArgsZ>;

const retrieveArgsZ = z.union([singleRetrieveArgsZ, multiRetrieveArgsZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

type RetrieveSchemas<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> = {
  schemas?: Schemas<Type, Config, StatusData>;
};

const retrieveResZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas?: Schemas<Type, Config, StatusData>,
) =>
  z.object({
    tasks: array.nullableZ(taskZ(schemas)),
  });

export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";
const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";
const COPY_ENDPOINT = "/hardware/task/copy";

const createReqZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas?: Schemas<Type, Config, StatusData>,
) => z.object({ tasks: newZ(schemas).array() });
const createResZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas?: Schemas<Type, Config, StatusData>,
) => z.object({ tasks: taskZ(schemas).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });
const copyResZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  schemas?: Schemas<Type, Config, StatusData>,
) => z.object({ task: taskZ(schemas) });

export class Client {
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

  async create(task: New): Promise<Task>;
  async create(tasks: New[]): Promise<Task[]>;
  async create(task: New | New[]): Promise<Task | Task[]>;

  async create<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    task: New<Type, Config>,
    schemas: Schemas<Type, Config, StatusData>,
  ): Promise<Task<Type, Config, StatusData>>;
  async create<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    tasks: New<Type, Config>[],
    schemas: Schemas<Type, Config, StatusData>,
  ): Promise<Task<Type, Config, StatusData>[]>;

  async create<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    task: New<Type, Config> | Array<New<Type, Config>>,
    schemas?: Schemas<Type, Config, StatusData>,
  ): Promise<Task<Type, Config, StatusData> | Array<Task<Type, Config, StatusData>>> {
    const isSingle = !Array.isArray(task);
    const createReq = createReqZ(schemas);
    const createRes = createResZ(schemas);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { tasks: array.toArray(task) } as z.infer<typeof createReq>,
      createReq,
      createRes,
    );
    const sugared = this.sugar<Type, Config, StatusData>(
      res.tasks as Payload<Type, Config, StatusData>[],
      schemas,
    );
    return isSingle ? sugared[0] : sugared;
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

  async retrieve<
    Type extends z.ZodLiteral<string>,
    Config extends z.ZodType,
    StatusData extends z.ZodType,
  >(
    args: RetrieveSingleParams & RetrieveSchemas<Type, Config, StatusData>,
  ): Promise<Task<Type, Config, StatusData>>;
  async retrieve(args: RetrieveSingleParams): Promise<Task>;
  async retrieve<
    Type extends z.ZodLiteral<string>,
    Config extends z.ZodType,
    StatusData extends z.ZodType,
  >(
    args: RetrieveMultipleParams & RetrieveSchemas<Type, Config, StatusData>,
  ): Promise<Task<Type, Config, StatusData>[]>;
  async retrieve(args: RetrieveMultipleParams): Promise<Task[]>;
  async retrieve<
    Type extends z.ZodLiteral<string>,
    Config extends z.ZodType,
    StatusData extends z.ZodType,
  >({
    schemas,
    ...args
  }: RetrieveArgs & RetrieveSchemas<Type, Config, StatusData>): Promise<
    Task<Type, Config, StatusData> | Task<Type, Config, StatusData>[]
  > {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ(schemas),
    );
    const tasks = res.tasks as Payload<Type, Config, StatusData>[];
    const sugared = this.sugar<Type, Config, StatusData>(tasks, schemas);
    checkForMultipleOrNoResults("Task", args, sugared, isSingle);
    return isSingle ? sugared[0] : sugared;
  }

  async copy(key: string, name: string, snapshot: boolean): Promise<Task> {
    const copyRes = copyResZ();
    const response = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyRes,
    );
    return this.sugar(response.task as Payload);
  }

  async retrieveSnapshottedTo(taskKey: Key): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null) throw NOT_CREATED_ERROR;
    return await retrieveSnapshottedTo(taskKey, this.ontologyClient);
  }

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    payloads: Payload<Type, Config, StatusData>[],
    schemas?: Schemas<Type, Config, StatusData>,
  ): Task<Type, Config, StatusData>[];

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    payload: Payload<Type, Config, StatusData>,
    schemas?: Schemas<Type, Config, StatusData>,
  ): Task<Type, Config, StatusData>;

  sugar<
    Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
    Config extends z.ZodType = z.ZodType,
    StatusData extends z.ZodType = z.ZodType,
  >(
    payloads: Payload<Type, Config, StatusData> | Payload<Type, Config, StatusData>[],
    schemas?: Schemas<Type, Config, StatusData>,
  ): Task<Type, Config, StatusData>[] | Task<Type, Config, StatusData> {
    const isSingle = !Array.isArray(payloads);
    const res = array.toArray(payloads).map(
      ({ key, name, type, config, status, internal, snapshot }) =>
        new Task(
          {
            key,
            name,
            type,
            config,
            internal,
            snapshot,
            status,
          },
          schemas,
          this.frameClient,
          this.ontologyClient,
          this.rangeClient,
        ),
    );
    return isSingle ? res[0] : res;
  }

  async executeCommand(params: ExecuteCommandParams): Promise<string>;

  async executeCommand(params: ExecuteCommandsParams): Promise<string[]>;

  async executeCommand(
    params: ExecuteCommandParams | ExecuteCommandsParams,
  ): Promise<string | string[]> {
    if ("commands" in params)
      return await executeCommands({ ...params, frameClient: this.frameClient });
    return await executeCommand({ ...params, frameClient: this.frameClient });
  }

  async executeCommandSync<StatusData extends z.ZodType = z.ZodType>(
    parms: ExecuteCommandsSyncParams<StatusData>,
  ): Promise<Status<StatusData>[]>;

  async executeCommandSync<StatusData extends z.ZodType = z.ZodType>(
    parms: ExecuteCommandSyncParams<StatusData>,
  ): Promise<Status<StatusData>>;

  async executeCommandSync<StatusData extends z.ZodType = z.ZodType>(
    params:
      | ExecuteCommandsSyncParams<StatusData>
      | ExecuteCommandSyncParams<StatusData>,
  ): Promise<Status<StatusData> | Status<StatusData>[]> {
    if ("commands" in params) {
      const retrieveNames = async () => {
        const { commands } = params;
        const ts = await this.retrieve({ keys: commands.map((t) => t.task) });
        return ts.map((t) => t.name);
      };
      return await executeCommandsSync({
        ...params,
        frameClient: this.frameClient,
        name: retrieveNames,
      });
    }
    const retrieveName = async () => {
      const { task } = params;
      const t = await this.retrieve({ key: task });
      return t.name;
    };
    return await executeCommandSync({
      frameClient: this.frameClient,
      name: retrieveName,
      ...params,
    });
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "task", key });

interface ExecuteCommandInternalParams {
  frameClient: framer.Client | null;
  task: Key;
  type: string;
  args?: {};
}

const executeCommand = async ({
  frameClient,
  task,
  type,
  args,
}: ExecuteCommandInternalParams): Promise<string> =>
  (await executeCommands({ frameClient, commands: [{ args, task, type }] }))[0];

export interface NewCommand {
  task: Key;
  type: string;
  args?: {};
}

interface ExecuteCommandsInternalParams {
  frameClient: framer.Client | null;
  commands: NewCommand[];
}

const executeCommands = async ({
  frameClient,
  commands,
}: ExecuteCommandsInternalParams): Promise<string[]> => {
  if (frameClient == null) throw NOT_CREATED_ERROR;
  const w = await frameClient.openWriter(COMMAND_CHANNEL_NAME);
  const cmds = commands.map((c) => ({ ...c, key: id.create() }));
  await w.write(COMMAND_CHANNEL_NAME, cmds);
  await w.close();
  return cmds.map((c) => c.key);
};

interface ExecuteCommandSyncInternalParams<StatusData extends z.ZodType = z.ZodType>
  extends Omit<ExecuteCommandsSyncInternalParams<StatusData>, "commands">,
    TaskExecuteCommandSyncParams {
  task: Key;
}

const executeCommandSync = async <StatusData extends z.ZodType = z.ZodType>({
  frameClient,
  task,
  type,
  timeout,
  name: taskName,
  statusDataZ,
  args,
}: ExecuteCommandSyncInternalParams<StatusData>): Promise<Status<StatusData>> =>
  (
    await executeCommandsSync({
      frameClient,
      commands: [{ args, task, type }],
      timeout,
      statusDataZ,
      name: taskName,
    })
  )[0];

interface ExecuteCommandsSyncInternalParams<StatusData extends z.ZodType = z.ZodType> {
  frameClient: framer.Client | null;
  commands: NewCommand[];
  timeout?: CrudeTimeSpan;
  statusDataZ: StatusData;
  name: string | string[] | (() => Promise<string | string[]>);
}

const executeCommandsSync = async <StatusData extends z.ZodType = z.ZodType>({
  frameClient,
  commands,
  timeout = TimeSpan.seconds(10),
  statusDataZ,
  name: taskName,
}: ExecuteCommandsSyncInternalParams<StatusData>): Promise<Status<StatusData>[]> => {
  if (frameClient == null) throw NOT_CREATED_ERROR;
  const streamer = await frameClient.openStreamer(STATUS_CHANNEL_NAME);
  const cmdKeys = await executeCommands({ frameClient, commands });
  const parsedTimeout = new TimeSpan(timeout);
  let states: Status<StatusData>[] = [];
  let timeoutID: NodeJS.Timeout | undefined;
  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutID = setTimeout(() => {
      void (async () => {
        const taskKeys = commands.map((c) => c.task);
        reject(await formatTimeoutError("command", taskName, parsedTimeout, taskKeys));
      })();
    }, parsedTimeout.milliseconds);
  });
  try {
    while (true) {
      const frame = await Promise.race([streamer.read(), timeoutPromise]);
      const state = statusZ(statusDataZ).parse(frame.at(-1)[STATUS_CHANNEL_NAME]);
      if (!cmdKeys.includes(state.key)) continue;
      states = [...states.filter((s) => s.key !== state.key), state];
      if (states.length === cmdKeys.length) return states;
    }
  } finally {
    clearTimeout(timeoutID);
    streamer.close();
  }
};

const formatTimeoutError = async (
  type: string,
  name: string | string[] | (() => Promise<string | string[]>),
  timeout: TimeSpan,
  key: Key | Key[],
): Promise<Error> => {
  const formattedType = caseconv.capitalize(type);
  const formattedTimeout = timeout.toString();
  try {
    let names: string[];
    if (typeof name === "string") names = [name];
    else if (Array.isArray(name)) names = name;
    else names = array.toArray(await name());
    const formattedName = strings.naturalLanguageJoin(names);
    return new Error(
      `${formattedType} command to ${formattedName} timed out after ${formattedTimeout}`,
    );
  } catch (e) {
    console.error("Failed to retrieve task name for timeout error:", e);
    return new Error(
      `${formattedType} command to task with key ${strings.naturalLanguageJoin(key)} timed out after ${formattedTimeout}`,
    );
  }
};
