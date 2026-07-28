// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { query } from "@/query";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ontologyID,
  type Payload,
  payloadZ,
  type Status,
  statusZ,
} from "@/rack/types.gen";
import { status } from "@/status";
import { type task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_rack_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  integration: z.string().optional(),
  searchTerm: z.string().optional(),
  embedded: z.boolean().optional(),
  hostIsNode: z.boolean().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const retrieveResZ = z.object({ racks: payloadZ.array().default(() => []) });
export const rackZ = payloadZ;

const singleRetrieveParamsZ = z.union([
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
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

const multiRetrieveParamsZ = retrieveReqZ;

export type RetrieveMultipleParams = z.input<typeof multiRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, multiRetrieveParamsZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

type SingleQuery =
  { key: Key; includeStatus?: boolean } | { name: string; includeStatus?: boolean };

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset", "hostIsNode"] as const;

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = (params: RetrieveSingleParams): SingleQuery => {
  const base = "key" in params ? { key: params.key } : { name: params.name };
  return params.includeStatus === true ? { ...base, includeStatus: true } : base;
};

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...rack }: Payload): Omit<Payload, "status"> => rack;

const isSingleParams = (params: unknown): params is RetrieveSingleParams =>
  typeof params === "object" &&
  params !== null &&
  ("key" in params || "name" in params);

/**
 * Client-side matching for a request: key and name sets and payload-held
 * flags. Server-computed shapes (search, pagination, host resolution) never
 * reach this filter; they refetch instead.
 */
const requestFilter = (
  req: RetrieveRequest,
): ((r: Omit<Payload, "status">) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  return (r) => {
    if (keySet != null && !keySet.has(r.key)) return false;
    if (nameSet != null && !nameSet.has(r.name)) return false;
    if (req.integration != null && !r.integrations.includes(req.integration))
      return false;
    if (req.embedded != null && r.embedded !== req.embedded) return false;
    return true;
  };
};

// Rack statuses live in the status table under the "rack:<key>" status key,
// carrying the rack key in their details.
const affectedRackKeys = (
  event: query.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  if (event.variant === "set") {
    const parsed = statusZ.safeParse(event.value);
    return parsed.success ? [parsed.data.details.rack] : null;
  }
  const [type, key] = event.key.split(":");
  if (type !== "rack") return null;
  const parsed = keyZ.safeParse(Number(key));
  return parsed.success ? [parsed.data] : null;
};

export interface ClientConfig {
  unary: UnaryClient;
  tasks: task.Client;
  cache: query.Cache;
  statusStore: query.Table<status.Key, status.Status>;
  ontologyStores: ontology.Stores;
}

export class Client extends query.Retriever<
  typeof retrieveReqZ,
  Key,
  Omit<Payload, "status">,
  Rack,
  RetrieveSingleParams
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Omit<Payload, "status">>;

  constructor(cfg: ClientConfig) {
    const { cache, statusStore } = cfg;
    const store = cache.createTable<Key, Omit<Payload, "status">>({
      name: "racks",
      fetch: async (keys) => (await this.fetchThrough({ keys })).map(stripStatus),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, payloadZ, {
          value: ({ status: _, ...rack }) => rack,
        }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const statusWatch = <Q extends { includeStatus?: boolean }>() =>
      query.watch(statusStore, (event, q: Q) => {
        if (q.includeStatus !== true) return null;
        return affectedRackKeys(event);
      });
    const single = cache.queries<SingleQuery, Rack, Key, Omit<Payload, "status">>({
      name: "rack",
      table: store,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: ([record], q) => this.compose(record, q.includeStatus === true),
      keyOf: (q) => ("key" in q ? q.key : null),
      matches: (r, q) => ("key" in q ? r.key === q.key : r.name === q.name),
      single: true,
      watch: [statusWatch<SingleQuery>()],
    });
    super(cache, {
      name: "rack",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (r, req) => requestFilter(req)(r),
        serverFields: SERVER_FIELDS,
        watch: [statusWatch<RetrieveRequest>()],
      },
      compose: (record, q) =>
        this.compose(record, (q as { includeStatus?: boolean }).includeStatus === true),
      single: {
        is: isSingleParams,
        normalize: normalizeSingle,
        space: single as query.Retrieves<query.Params, Rack>,
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      ontology.deleteCachedResources(this.cfg.ontologyStores, ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/rack/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      ontology.renameCachedResource(this.cfg.ontologyStores, ontologyID(key), name),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...rename());
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      const r = await this.retrieve({ key });
      await this.create({ ...r.payload, name });
    });
    rename();
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const res = await this.cfg.unary.send(
      "/rack/create",
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.racks.map(stripStatus));
    const sugared = this.sugar(res.racks);
    return isSingle ? sugared[0] : sugared;
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(
        ({ key, name, status, integrations, taskCounter, embedded }) =>
          new Rack(
            key,
            name,
            this.cfg.tasks,
            status,
            integrations,
            taskCounter,
            embedded,
          ),
      );
    return isSingle ? sugared[0] : sugared;
  }

  /** Rebuilds a cached rack, attaching its cached status when requested. */
  private compose(cached: Omit<Payload, "status">, includeStatus: boolean): Rack {
    if (!includeStatus) return this.sugar(cached);
    const st = this.latestStatusOf(cached.key);
    if (st == null) return this.sugar(cached);
    return this.sugar({ ...cached, status: st });
  }

  // A rack's status may live under the "rack:<key>" row or under any status
  // whose details reference the rack; the freshest wins.
  private latestStatusOf(key: Key): Status | undefined {
    const rackKey = statusKey(key);
    const candidates = this.cfg.statusStore
      .get((s) => s.key === rackKey || status.detailsOf(s)?.rack === key)
      .map((s) => statusZ.safeParse(s))
      .filter((p) => p.success)
      .map((p) => p.data);
    if (candidates.length === 0) return undefined;
    return candidates.reduce((latest, s) =>
      new TimeStamp(s.time).afterEq(new TimeStamp(latest.time)) ? s : latest,
    );
  }

  /** Writes fetched racks and their included statuses. */
  private writeThrough(racks: Payload[]): void {
    this.store.set(racks.map(stripStatus));
    racks.forEach(({ status: st }) => {
      if (st != null) this.cfg.statusStore.set(st);
    });
  }

  private async execRetrieve(params: RetrieveParams): Promise<Payload[]> {
    const res = await this.cfg.unary.send(
      "/rack/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.racks;
  }

  /** Fetches racks and writes their included statuses through the caches. */
  private async fetchThrough(req: RetrieveRequest): Promise<Payload[]> {
    const racks = await this.execRetrieve(req);
    this.writeThrough(racks);
    return racks;
  }

  private async fetchSingle(q: SingleQuery): Promise<Rack> {
    // Names are not unique, so only key queries can be served from the table.
    // A status-bearing hit needs both the record and its status cached.
    if ("key" in q && q.includeStatus !== true) {
      const cached = this.store.get(q.key);
      if (cached != null) return this.sugar(cached);
    }
    const racks = await this.execRetrieve(q);
    checkForMultipleOrNoResults("Rack", q, racks, true);
    this.writeThrough(racks);
    return this.sugar(racks[0]);
  }
}

export class Rack {
  key: Key;
  name: string;
  status?: Status;
  integrations: string[];
  taskCounter: number;
  embedded: boolean;
  private readonly tasks: task.Client;

  constructor(
    key: Key,
    name: string,
    taskClient: task.Client,
    status?: Status,
    integrations: string[] = [],
    taskCounter: number = 0,
    embedded: boolean = false,
  ) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.status = status;
    this.integrations = integrations;
    this.taskCounter = taskCounter;
    this.embedded = embedded;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve({ rack: this.key });
  }

  async createTask(task: task.New): Promise<task.Task>;
  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas: Schemas,
  ): Promise<task.Task<Schemas>>;

  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas?: Schemas,
  ): Promise<task.Task<Schemas>> {
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
    return await this.tasks.create(task, schemas as Required<Schemas>);
  }

  async deleteTask(task: task.Key): Promise<void> {
    await this.tasks.delete([task]);
  }

  get payload(): Payload {
    return {
      key: this.key,
      name: this.name,
      status: this.status,
      integrations: this.integrations,
      taskCounter: this.taskCounter,
      embedded: this.embedded,
    };
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
