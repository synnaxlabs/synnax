// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive } from "@synnaxlabs/x";
import z from "zod";

import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { matchLabeledBy } from "@/label/store";
import { type Key, keyZ, type Label, labelZ, type New } from "@/label/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_label_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

const createReqZ = z.object({ labels: labelZ.array() });
const createResZ = z.object({ labels: labelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setReqZ = z.object({
  id: ontology.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  for: ontology.idZ.optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  relationships: query.Table<string, ontology.Relationship>;
}

export class Client extends query.Retriever<typeof retrieveRequestZ, Key, Label> {
  readonly type: string = "label";
  /** The label record table; injected into sibling clients at wiring. */
  readonly store: query.Table<Key, Label>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache, relationships } = cfg;
    const store = cache.createTable<Key, Label>({
      name: "labels",
      fetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, labelZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "label",
      table: store,
      request: {
        schema: retrieveRequestZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (label, req) => this.requestFilter(req)(label),
        watch: [
          query.watch(relationships, (event, req: RetrieveRequest) => {
            if (req.for == null) return null;
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!matchLabeledBy(rel, req.for)) return null;
            return [rel.to.key];
          }),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async label(
    id: ontology.ID,
    labels: Key[],
    opts: SetOptions & query.WriteOptions = {},
  ): Promise<void> {
    const rollback = new destructor.Chain();
    if (opts.replace === true)
      rollback.add(this.cfg.relationships.delete((r) => matchLabeledBy(r, id)));
    labels.forEach((key) => {
      const rel = labeledByRel(id, key);
      rollback.add(this.cfg.relationships.set(ontology.relationshipToString(rel), rel));
    });
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/label/set",
          { id, labels, replace: opts.replace },
          setReqZ,
          emptyResZ,
        ),
    );
  }

  async remove(
    id: ontology.ID,
    labels: Key[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const rollback = new destructor.Chain();
    rollback.add(
      this.cfg.relationships.delete(
        (r) => matchLabeledBy(r, id) && labels.includes(r.to.key),
      ),
    );
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/label/remove",
          { id, labels },
          removeReqZ,
          emptyResZ,
        ),
    );
  }

  async create(label: New, opts?: query.WriteOptions<Label[]>): Promise<Label>;
  async create(labels: New[], opts?: query.WriteOptions<Label[]>): Promise<Label[]>;
  async create(
    labels: New | New[],
    opts: query.WriteOptions<Label[]> = {},
  ): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const optimistic = array.toArray(labels).map((l) => labelZ.parse(l));
    const rollback = new destructor.Chain();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/label/create",
          { labels: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.store.delete(keysArr),
      this.cfg.relationships.delete(
        (r) =>
          r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
          r.to.type === "label" &&
          keysArr.includes(r.to.key),
      ),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/label/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    drop();
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Label[]> {
    const res = await this.cfg.unary.send(
      "/label/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.labels;
  }

  /**
   * Client-side matching for a request: key sets, names, and labeled-entity
   * membership via the relationship table. Server-computed shapes (search,
   * limit/offset) never reach this filter; they refetch instead.
   */
  private requestFilter(req: RetrieveRequest): (l: Label) => boolean {
    const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
    const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
    return (l) => {
      if (keySet != null && !keySet.has(l.key)) return false;
      if (nameSet != null && !nameSet.has(l.name)) return false;
      if (req.for != null && !this.isLabelOf(req.for, l.key)) return false;
      return true;
    };
  }

  private isLabelOf(id: ontology.ID, key: Key): boolean {
    return this.cfg.relationships.has(
      ontology.relationshipToString(labeledByRel(id, key)),
    );
  }
}

const retrieveResponseZ = z.object({ labels: labelZ.array().default(() => []) });

const labeledByRel = (id: ontology.ID, key: Key): ontology.Relationship => ({
  from: id,
  type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  to: ontologyID(key),
});

export const ontologyID = ontology.createIDFactory<Key>("label");
export const TYPE_ONTOLOGY_ID = ontologyID("");
