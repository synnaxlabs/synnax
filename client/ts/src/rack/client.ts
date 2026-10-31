// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, deep } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { query } from "@/query";
import { type Condition } from "@/rack/builder.gen";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ontologyID,
  type Payload,
  payloadZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Status,
  statusZ,
} from "@/rack/types.gen";
import { LINES } from "@/rack/where";
import { matches, serverComputed } from "@/rack/where.gen";
import { type status } from "@/status";
import { type task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_rack_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), retrieveRequestZ]);
export const rackZ = payloadZ;

const singleRetrieveParamsZ = z.union([
  z
    .object({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }): RetrieveRequest => ({
      where: { key: { eq: key } },
      includeStatus,
    })),
  z
    .object({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }): RetrieveRequest => ({
      where: { name: { eq: name } },
      includeStatus,
    })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

export type RetrieveMultipleParams = RetrieveRequest;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

type SingleQuery =
  { key: Key; includeStatus?: boolean } | { name: string; includeStatus?: boolean };

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = (params: RetrieveSingleParams): SingleQuery => {
  const base = "key" in params ? { key: params.key } : { name: params.name };
  return params.includeStatus === true ? { ...base, includeStatus: true } : base;
};

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    z.strictObject({ name: z.string(), includeStatus: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...rack }: Payload): Omit<Payload, "status"> => rack;

// Rack statuses live in the status table under the "rack:<key>" status key.
const affectedRackKeys = (
  event: query.TableEvent<status.Key, status.Status>,
): Key[] | null => {
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
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  Condition,
  typeof retrieveMultiParamsZ,
  Key,
  Omit<Payload, "status">,
  Rack,
  RetrieveSingleParams,
  SingleQuery
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Omit<Payload, "status">>;

  constructor(cfg: ClientConfig) {
    const { cache, statusStore } = cfg;
    const store = cache.createTable<Key, Omit<Payload, "status">>({
      name: "racks",
      fetch: async (keys) =>
        (await this.fetchThrough(query.keysRequest(keys))).map(stripStatus),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, payloadZ, {
          value: ({ status: _, ...rack }) => rack,
        }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    // Composed rows always carry status; includeStatus only gates fetch flags.
    const composed = cache.derive<Key, Omit<Payload, "status">, Rack>({
      name: "rack.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      watch: [query.deriveWatch(statusStore, (event) => affectedRackKeys(event))],
    });
    const single = cache.queries<SingleQuery, Rack, Key, Rack>({
      name: "rack",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => ("key" in q ? q.key : null),
      matches: (r, q) => ("key" in q ? r.key === q.key : r.name === q.name),
      single: true,
    });
    super(cache, {
      name: "rack",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (r, req) => matches(LINES, r, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        watch: [
          query.watch(statusStore, (event, q: RetrieveRequest) => {
            if (q.includeStatus !== true) return null;
            return affectedRackKeys(event);
          }),
        ],
      },
      compose: (record, q) =>
        this.compose(record, (q as { includeStatus?: boolean }).includeStatus === true),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/rack/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send("/rack/rename", { key, name }, renameReqZ, emptyResZ),
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
        ({ key, name, status, integrations, embedded }) =>
          new Rack(key, name, this.cfg.tasks, status, integrations, embedded),
      );
    return isSingle ? sugared[0] : sugared;
  }

  private compose(cached: Omit<Payload, "status">, includeStatus: boolean): Rack {
    if (!includeStatus) return this.sugar(cached);
    const st = this.statusOf(cached.key);
    if (st == null) return this.sugar(cached);
    return this.sugar({ ...cached, status: st });
  }

  // Only the "rack:<key>" row is the rack's status. Task statuses name their rack in
  // their details, and matching on that would let them take the rack's status over.
  private statusOf(key: Key): Status | undefined {
    const parsed = statusZ.safeParse(this.cfg.statusStore.get(statusKey(key)));
    return parsed.success ? parsed.data : undefined;
  }

  private writeThrough(racks: Payload[]): void {
    this.store.ingest(racks.map(stripStatus));
    racks.forEach(({ status: st }) => {
      if (st != null) this.cfg.statusStore.ingest(st);
    });
  }

  private async execRetrieve(params: RetrieveParams): Promise<Payload[]> {
    const res = await this.cfg.unary.send(
      "/rack/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.racks;
  }

  private async fetchThrough(req: RetrieveRequest): Promise<Payload[]> {
    const racks = await this.execRetrieve(req);
    this.writeThrough(racks);
    return racks;
  }

  private async fetchSingle(q: SingleQuery): Promise<Rack> {
    // Names are not unique and the table never holds status, so only keyed status-free
    // queries can be served from it.
    if ("key" in q && q.includeStatus !== true) {
      const racks = await this.store.retrieve([q.key]);
      checkForMultipleOrNoResults("Rack", q, racks, true);
      return this.sugar(racks[0]);
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
  embedded: boolean;
  private readonly tasks: task.Client;

  constructor(
    key: Key,
    name: string,
    taskClient: task.Client,
    status?: Status,
    integrations: string[] = [],
    embedded: boolean = false,
  ) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.status = status;
    this.integrations = integrations;
    this.embedded = embedded;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve({ where: { rack: { eq: this.key } } });
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
    task.rack = this.key;
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
      embedded: this.embedded,
    };
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
