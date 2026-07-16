// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { type cache } from "@/cache";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { type Key, keyZ } from "@/status/payload";
import { bindStore, STORE_KEY } from "@/status/store";
import { type New, type Status, statusZ } from "@/status/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const setReqZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    parent: ontology.idZ.optional(),
    statuses: statusZ({ details: detailsSchema }).array(),
  });
const setResZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => z.object({ statuses: statusZ({ details: detailsSchema }).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  includeLabels: z.boolean().optional(),
  hasLabels: label.keyZ.array().optional(),
  variants: z.string().array().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ, includeLabels: z.boolean().optional() })
  .transform(({ key, includeLabels }) => ({ keys: [key], includeLabels }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;
export type MultiRetrieveParams = z.input<typeof retrieveRequestZ>;

const retrieveResponseZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    statuses: statusZ({ details: detailsSchema })
      .array()
      .default(() => []),
  });

export interface SetOptions {
  parent?: ontology.ID;
}

export class Client {
  readonly type: string = "status";
  private readonly client: UnaryClient;
  private readonly store_?: cache.Store<Key, Status>;

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine);
    this.store_ = engine.store(STORE_KEY);
  }

  /**
   * Read surface of the status cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, Status> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  async retrieve<DetailsSchema extends z.ZodType>(
    params: SingleRetrieveParams & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async retrieve(params: SingleRetrieveParams): Promise<Status>;
  async retrieve(params: MultiRetrieveParams): Promise<Status[]>;
  async retrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    params: RetrieveParams & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isSingle = "key" in params;
    const res = await this.client.send(
      "/status/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ<DetailsSchema>(params.detailsSchema),
    );
    checkForMultipleOrNoResults("Status", params, res.statuses, isSingle);
    const statuses = res.statuses as Status<DetailsSchema>[];
    return isSingle ? statuses[0] : statuses;
  }

  async set<DetailsSchema extends z.ZodType>(
    status: New<DetailsSchema>,
    opts?: SetOptions & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async set(status: New, opts?: SetOptions): Promise<Status>;
  async set(statuses: New[], opts?: SetOptions): Promise<Status[]>;
  async set<DetailsSchema extends z.ZodType = z.ZodNever>(
    statuses: New<DetailsSchema> | New<DetailsSchema>[],
    opts: SetOptions & { detailsSchema?: DetailsSchema } = {},
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isMany = Array.isArray(statuses);
    const res = await this.client.send(
      "/status/set",
      {
        statuses: array.toArray(statuses) as z.input<
          ReturnType<typeof setReqZ<DetailsSchema>>
        >["statuses"],
        parent: opts.parent,
      },
      setReqZ(opts.detailsSchema),
      setResZ(opts.detailsSchema),
    );
    const created = res.statuses as Status<DetailsSchema>[];
    return isMany ? created : created[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/status/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
