// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  array,
  type destructor,
  filter,
  scheduler,
  TimeSpan,
  uuid,
} from "@synnaxlabs/x";
import z from "zod";

import { type batch } from "@/batch";
import { type label } from "@/label";
import { type ontology } from "@/ontology";
import { query } from "@/query";
import { resource } from "@/resource";
import { DELETE_CHANNEL_NAME, SET_CHANNEL_NAME } from "@/status/payload";
import {
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  type Status,
  statusZ,
} from "@/status/types.gen";
import { LINES } from "@/status/where";
import { matches, retrieveInputZ, serverComputed } from "@/status/where.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const setReqZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => z.object({ statuses: statusZ({ details: detailsSchema }).array() });
const setResZ = setReqZ;
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });

export type RetrieveMultipleParams = RetrieveRequest;

const filterParamsZ = retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape);
type FilterParams = query.RequestInput<typeof filterParamsZ>;
const requestParamsZ = filter.requestInputZ(retrieveRequestZ, filterParamsZ);

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), requestParamsZ]);

const singleRetrieveParamsZ = z
  .strictObject({ key: keyZ })
  .transform(({ key }): RetrieveRequest => ({ where: { key: { eq: key } } }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, requestParamsZ]);

export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;
type OneParams = query.OneInput<typeof retrieveMultiParamsZ>;

const isOneParams = (params: object): params is SingleRetrieveParams | OneParams =>
  query.addressesOne(params);
export type RetrieveParams = SingleRetrieveParams | RetrieveMultipleParams;

const retrieveResponseZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    statuses: statusZ({ details: detailsSchema })
      .array()
      .default(() => []),
  });

const ownerOf = (s: Status): string | null =>
  s.owner == null ? null : resource.idToString(s.owner);

const OWNED_FETCH_DEBOUNCE = TimeSpan.milliseconds(10);

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Status,
  Status,
  SingleRetrieveParams
> {
  readonly type: string = "status";
  readonly store: query.Table<Key, Status>;
  private readonly cfg: ClientConfig;
  private readonly owners: query.LookupIndex<Key, Status, string>;
  private readonly writeHandlers = new Set<(status: Status) => void>();
  private readonly ownedFetches = new scheduler.Batcher<resource.ID[], Status[]>({
    interval: OWNED_FETCH_DEBOUNCE,
    exec: async (requests) => {
      const owners = requests.flatMap(({ req }) => req);
      const since = this.store.stamp();
      const statuses = await this.execRetrieve({ where: { owners } });
      this.store.ingest(statuses, { since });
      requests.forEach(({ req, resolve }) => {
        const wanted = new Set(resource.idToString(req));
        resolve(statuses.filter((s) => wanted.has(ownerOf(s) ?? "")));
      });
    },
  });

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const owners = new query.LookupIndex<Key, Status, string>(ownerOf);
    const store = cache.createTable<Key, Status>({
      name: "statuses",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, statusZ()),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
      indexes: [owners],
    });
    super(cache, {
      name: "status",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (status, req) => matches(LINES, status, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.owners = owners;
    cache.listen({
      channel: SET_CHANNEL_NAME,
      schema: statusZ(),
      onChange: (written) =>
        written.forEach((status) => this.writeHandlers.forEach((h) => h(status))),
    });
  }

  async retrieve<DetailsSchema extends z.ZodType>(
    params: (SingleRetrieveParams | OneParams) & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async retrieve(params: Key | SingleRetrieveParams | OneParams): Promise<Status>;
  async retrieve(
    params: Key[] | query.RequestInput<typeof retrieveMultiParamsZ>,
  ): Promise<Status[]>;
  async retrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    rawParams:
      | Key
      | Key[]
      | ((RetrieveParams | OneParams | FilterParams) & {
          detailsSchema?: DetailsSchema;
        }),
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    if (Array.isArray(rawParams))
      return (await super.retrieve(rawParams)) as Status<DetailsSchema>[];
    const params = typeof rawParams === "string" ? { key: rawParams } : rawParams;
    const { detailsSchema, ...rest } = params;
    const isSingle = isOneParams(rest);
    // Schema-parametrized retrieves validate details for one caller; their
    // results are not shared through the cache.
    if (detailsSchema != null) {
      const request = query.dropOne(rest);
      const statuses = await this.execRetrieve(request, detailsSchema);
      checkForMultipleOrNoResults(
        "status",
        retrieveParamsZ.parse(request),
        statuses,
        isSingle,
      );
      return isSingle ? statuses[0] : statuses;
    }
    if (isSingle) return (await super.retrieve(rest)) as Status<DetailsSchema>;
    return (await super.retrieve(rest)) as Status<DetailsSchema>[];
  }

  /**
   * Writes statuses to the Core. The Core writes an owned status to the status its
   * owner holds, whatever key it carries. A create stores the labels a status carries,
   * and an update keeps the stored labels: change them with the label client.
   * @returns each status as the Core stores it.
   */
  async set<DetailsSchema extends z.ZodType>(
    status: New<DetailsSchema>,
    opts?: SetOptions & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async set(status: New, opts?: SetOptions): Promise<Status>;
  async set(statuses: New[], opts?: SetOptions): Promise<Status[]>;
  async set(statuses: New | New[], opts?: SetOptions): Promise<Status | Status[]>;
  async set<DetailsSchema extends z.ZodType = z.ZodNever>(
    statuses: New<DetailsSchema> | New<DetailsSchema>[],
    opts: SetOptions & { detailsSchema?: DetailsSchema } = {},
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isMany = Array.isArray(statuses);
    const { endpoint, payload, apply } = this.setRequest(statuses, opts);
    const res = await query.optimistic({
      rollbacks: [apply()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          endpoint,
          payload,
          setReqZ(opts.detailsSchema),
          setResZ(opts.detailsSchema),
        ),
    });
    const created = res.statuses as Status<DetailsSchema>[];
    this.store.set(created);
    return isMany ? created : created[0];
  }

  /**
   * Builds a request that writes statuses, as set does. Its cache write holds each
   * status whose stored form the client can predict.
   */
  setRequest<DetailsSchema extends z.ZodType = z.ZodNever>(
    statuses: New<DetailsSchema> | New<DetailsSchema>[],
    opts: { detailsSchema?: DetailsSchema } = {},
  ): batch.Request<z.input<ReturnType<typeof setReqZ<DetailsSchema>>>> & {
    apply: () => destructor.Destructor;
  } {
    // Filling the schema defaults up front gives every status a key, so the cache can
    // hold it before the Core answers. Status keys are UUIDs, not the short id that the
    // schema default makes.
    const schema = statusZ<DetailsSchema>({ details: opts.detailsSchema });
    const parsed = array.toArray(statuses).map(
      (status) =>
        schema.parse({
          ...status,
          key: status.key ?? uuid.create(),
        }) as Status<DetailsSchema>,
    );
    const predicted = parsed.map((status) => this.predict(status));
    const sent = predicted.map((status, i) => status ?? parsed[i]);
    return {
      endpoint: "/status/set",
      payload: {
        statuses: sent as z.input<
          ReturnType<typeof setReqZ<DetailsSchema>>
        >["statuses"],
      },
      apply: () => this.store.set(predicted.filter((status) => status != null)),
    };
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(resourceID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/status/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  /**
   * Changes the labels of the cached status under the key with update, and returns the
   * rollback. It skips a status the cache does not hold.
   */
  relabel(
    key: Key,
    update: (labels: label.Key[]) => label.Key[],
  ): destructor.Destructor {
    return this.store.set(key, (p) =>
      p == null ? undefined : { ...p, labels: update(p.labels) },
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/status/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every status set delivered to the cache. */
  onSet(handler: (status: Status) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /**
   * Subscribes to each status write that the Core streams to this client, from any
   * writer, and to each {@link setLocal} status. Unlike {@link onSet}, it skips a
   * status that reaches the cache another way, such as through a retrieve.
   */
  onWrite(handler: (status: Status) => void): destructor.Destructor {
    this.writeHandlers.add(handler);
    return () => this.writeHandlers.delete(handler);
  }

  /**
   * Sets a status that this client makes and the Core never holds, such as a stand-in
   * reply to a command. The status reaches the cache and the {@link onWrite} handlers.
   */
  setLocal(status: Status): void {
    this.store.set(status);
    this.writeHandlers.forEach((h) => h(status));
  }

  /** Subscribes to every status delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  /**
   * Fetches the statuses that the resources own into the cache. Calls made in the same
   * few milliseconds share one request.
   * @returns the statuses the resources own.
   */
  async retrieveOwned(owners: resource.ID[]): Promise<Status[]> {
    return await this.ownedFetches.enqueue(owners);
  }

  /**
   * @returns the cached status that the resource owns, or undefined when the cache
   * holds none.
   */
  ownedBy(owner: resource.ID): Status | undefined {
    return this.owners.get(resource.idToString(owner))[0];
  }

  /**
   * @returns the keys of the owners of the given type whose status the event sets or
   * deletes, or null when it touches none. An evicted status names no owner.
   */
  ownerKeys(
    event: query.TableEvent<Key, Status>,
    type: resource.Type,
  ): string[] | null {
    const changed =
      event.variant === "set"
        ? event.value
        : this.store.getTombstone(event.key)?.corpse;
    if (changed?.owner == null || changed.owner.type !== type) return null;
    return [changed.owner.key];
  }

  /**
   * @returns the status as the Core will store it, or null when the status has an
   * owner and the cache does not hold the key of the status the owner holds.
   */
  private predict<D extends z.ZodType>(status: Status<D>): Status<D> | null {
    let { key } = status;
    if (status.owner != null) {
      const held = this.ownedBy(status.owner);
      if (held == null) return null;
      key = held.key;
    }
    const prev = this.store.get(key);
    return { ...status, key, labels: prev?.labels ?? status.labels };
  }

  private async execRetrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    params: RetrieveParams | OneParams | FilterParams,
    detailsSchema?: DetailsSchema,
  ): Promise<Status<DetailsSchema>[]> {
    const res = await this.cfg.unary.send(
      "/status/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ<DetailsSchema>(detailsSchema),
    );
    return res.statuses as Status<DetailsSchema>[];
  }
}

export interface SetOptions extends query.WriteOptions {}
