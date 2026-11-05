// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { createOf, encode, reduceAll } from "@/label/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/label/actions.gen";
import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { matchLabeledBy } from "@/label/store";
import {
  type Key,
  keyZ,
  type Label,
  labelZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Where,
} from "@/label/types.gen";
import { type Lines, matches, serverComputed } from "@/label/where.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_label_action_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

const createReqZ = z.object({ labels: labelZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ labels: labelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setReqZ = z.object({
  id: ontology.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
/** Options for `Client.set`. Set `replace` to swap the whole label set at once. */
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), retrieveRequestZ]);

/** @returns the resources that the for lines of where name. */
const forIDs = (where: Where): ontology.ID[] => [
  ...(where.for != null ? [where.for] : []),
  ...[...(where.and ?? []), ...(where.or ?? [])].flatMap(forIDs),
  ...(where.not != null ? forIDs(where.not) : []),
];

/** @returns the resource whose labels a request selects, and nothing else, or null. */
const labelsOf = ({ where, ...rest }: RetrieveRequest): ontology.ID | null => {
  if (where?.for == null || Object.values(rest).some((v) => v != null)) return null;
  const { for: id, ...others } = where;
  return Object.values(others).every((v) => v == null) ? id : null;
};

/** Names one label. */
export type RetrieveSingleParams = { key: Key };
/** Everything a multi-label retrieval can filter on, including the resource labeled. */
export type RetrieveMultipleParams = RetrieveRequest;
/** Params for a label retrieval, single or multiple. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

/**
 * Creates, reads, deletes, and attaches labels on a Core. Reach it through
 * `client.labels`.
 */
export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Label> {
  readonly type: string = "label";
  /** The label record table; injected into sibling clients at wiring. */
  readonly store: query.Table<Key, Label>;
  private readonly cfg: ClientConfig;
  private readonly dispatcher: actions.Controller<Key, Label, Action>;

  constructor(cfg: ClientConfig) {
    const { cache, ontology: ontologyClient } = cfg;
    const { relationships } = ontologyClient.cache;
    // Resources whose every label the relationship table holds. A change stream gap
    // can hide a removal, so a new epoch forgets them.
    const labelSets = new Set<string>();
    cache.onEpoch(() => labelSets.clear());
    // Null when the client neither saw the relationship nor holds every label of id.
    const labeledBy = (id: ontology.ID, key: Key): boolean | null => {
      const status = relationships.status(
        ontology.relationshipToString(labeledByRel(id, key)),
      );
      if (status === "present") return true;
      if (status === "tombstoned" || labelSets.has(ontology.idToString(id)))
        return false;
      return null;
    };
    const lines: Lines = { for: (l, id) => labeledBy(id, l.key) === true };
    const store = cache.createTable<Key, Label>({
      name: "labels",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a label reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Label, Action>({
      store,
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      send: actions.sender(cfg.unary, "/label/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "label",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => {
          const id = labelsOf(req);
          const { epoch } = cache;
          const since = relationships.stamp();
          const labels = await this.execRetrieve(req);
          // A response from before a reconnect can miss changes that the stream missed.
          if (id == null || cache.epoch !== epoch) return labels;
          // The response holds every label of id, so it replaces the cached set.
          const fetched = new Map(
            labels.map((l) => {
              const rel = labeledByRel(id, l.key);
              return [ontology.relationshipToString(rel), rel];
            }),
          );
          const vanished = ontologyClient.cache
            .relationshipsFrom(id)
            .filter((r) => matchLabeledBy(r, id))
            .map(ontology.relationshipToString)
            .filter((key) => !fetched.has(key));
          relationships.batch(() => {
            relationships.ingestDeletes(vanished, since);
            fetched.forEach((rel, key) => relationships.ingest(key, rel, { since }));
          });
          labelSets.add(ontology.idToString(id));
          return labels;
        },
        matches: (l, req) => {
          if (
            req.where != null &&
            forIDs(req.where).some((id) => labeledBy(id, l.key) == null)
          )
            return null;
          return matches(lines, l, req.where);
        },
        serverComputed: (req) => serverComputed(lines, req.where),
        watch: [
          query.watch(relationships, (event, req: RetrieveRequest) => {
            if (req.where == null) return null;
            const ids = forIDs(req.where);
            if (ids.length === 0) return null;
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!ids.some((id) => matchLabeledBy(rel, id))) return null;
            return [rel.to.key];
          }),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async label(
    id: ontology.ID,
    labels: Key[],
    opts: SetOptions & query.WriteOptions = {},
  ): Promise<void> {
    await query.optimistic({
      rollbacks: [
        ...(opts.replace === true
          ? [this.cfg.ontology.cache.relationships.delete((r) => matchLabeledBy(r, id))]
          : []),
        ...labels.map((key) => {
          const rel = labeledByRel(id, key);
          return this.cfg.ontology.cache.relationships.set(
            ontology.relationshipToString(rel),
            rel,
          );
        }),
      ],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/label/set",
          { id, labels, replace: opts.replace },
          setReqZ,
          emptyResZ,
        ),
    });
  }

  async remove(
    id: ontology.ID,
    labels: Key[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    await query.optimistic({
      rollbacks: [
        this.cfg.ontology.cache.relationships.delete(
          (r) => matchLabeledBy(r, id) && labels.includes(r.to.key),
        ),
      ],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/label/remove",
          { id, labels },
          removeReqZ,
          emptyResZ,
        ),
    });
  }

  async create(label: New, opts?: query.WriteOptions<Label[]>): Promise<Label>;
  async create(labels: New[], opts?: query.WriteOptions<Label[]>): Promise<Label[]>;
  async create(
    labels: New | New[],
    opts: query.WriteOptions<Label[]> = {},
  ): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const optimistic = array
      .toArray(labels)
      .map((l) => zod.parse(labelZ, l, { label: "label" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/label/create",
              { labels: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  /**
   * Applies actions to the cached label and sends them to the Core. Returns false
   * without side effects when the label is not cached.
   * @throws {Error} if the label cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached label drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.store.delete(keysArr),
      this.cfg.ontology.cache.relationships.delete(
        (r) =>
          r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
          r.to.type === "label" &&
          keysArr.includes(r.to.key),
      ),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/label/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Label[]> {
    const res = await this.cfg.unary.send(
      "/label/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.labels;
  }
}

const labeledByRel = (id: ontology.ID, key: Key): ontology.Relationship => ({
  from: id,
  type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  to: ontologyID(key),
});

export const ontologyID = ontology.createIDFactory<Key>("label");
export const TYPE_ONTOLOGY_ID = ontologyID("");
