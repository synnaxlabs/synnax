// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { label } from "@/label";
import { ontology } from "@/ontology";
import { type Key, keyZ, type New, newZ, type Status, statusZ } from "@/status/payload";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const setReqZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    parent: ontology.idZ.optional(),
    statuses: newZ(detailsSchema).array(),
  });
const setResZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => z.object({ statuses: statusZ(detailsSchema).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  includeLabels: z.boolean().optional(),
  hasLabels: label.keyZ.array().optional(),
});

const singleRetrieveArgsZ = z
  .object({ key: keyZ, includeLabels: z.boolean().optional() })
  .transform(({ key, includeLabels }) => ({ keys: [key], includeLabels }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type SingleRetrieveArgs = z.input<typeof singleRetrieveArgsZ>;
export type MultiRetrieveArgs = z.input<typeof retrieveRequestZ>;

const retrieveResponseZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => z.object({ statuses: array.nullishToEmpty(statusZ(detailsSchema)) });

export interface SetOptions {
  parent?: ontology.ID;
}

export class Client {
  readonly type: string = "status";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve<DetailsSchema extends z.ZodType>(
    args: SingleRetrieveArgs & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async retrieve(args: SingleRetrieveArgs): Promise<Status>;
  async retrieve(args: MultiRetrieveArgs): Promise<Status[]>;
  async retrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    args: RetrieveArgs & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      "/status/retrieve",
      args,
      retrieveArgsZ,
      retrieveResponseZ<DetailsSchema>(args.detailsSchema),
    );
    checkForMultipleOrNoResults("Status", args, res.statuses, isSingle);
    const statuses = res.statuses as unknown as Status<DetailsSchema>[];
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
    const res = await sendRequired<
      ReturnType<typeof setReqZ<DetailsSchema>>,
      ReturnType<typeof setResZ<DetailsSchema>>
    >(
      this.client,
      "/status/set",
      {
        statuses: array.toArray(statuses),
        parent: opts.parent,
      },
      setReqZ(opts.detailsSchema),
      setResZ(opts.detailsSchema),
    );
    const created = res.statuses as unknown as Status<DetailsSchema>[];
    return isMany ? created : created[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      "/status/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
