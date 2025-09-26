// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x/array";
import z from "zod";

import { ontology } from "@/ontology";
import { type Key, keyZ, type New, newZ, type Status, statusZ } from "@/status/payload";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const setReqZ = z.object({
  parent: ontology.idZ.optional(),
  statuses: newZ.array(),
});
const setResZ = z.object({ statuses: statusZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const SET_ENDPOINT = "/status/set";
const DELETE_ENDPOINT = "/status/delete";
const RETRIEVE_ENDPOINT = "/status/retrieve";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
  includeLabels: z.boolean().optional(),
});

const singleRetrieveArgsZ = z
  .object({ key: keyZ, includeLabels: z.boolean().optional() })
  .transform(({ key, includeLabels }) => ({ keys: [key], includeLabels }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type SingleRetrieveArgs = z.input<typeof singleRetrieveArgsZ>;
export type MultiRetrieveArgs = z.input<typeof retrieveRequestZ>;

const retrieveResponseZ = z.object({ statuses: nullableArrayZ(statusZ) });

export interface SetOptions {
  parent?: ontology.ID;
}

export class Client {
  readonly type: string = "status";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(args: SingleRetrieveArgs): Promise<Status>;
  async retrieve(args: MultiRetrieveArgs): Promise<Status[]>;
  async retrieve(args: RetrieveArgs): Promise<Status | Status[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResponseZ,
    );
    checkForMultipleOrNoResults("Status", args, res.statuses, isSingle);
    return isSingle ? res.statuses[0] : res.statuses;
  }

  async set(status: New, opts?: SetOptions): Promise<Status>;
  async set(statuses: New[], opts?: SetOptions): Promise<Status[]>;
  async set(statuses: New | New[], opts: SetOptions = {}): Promise<Status | Status[]> {
    const isMany = Array.isArray(statuses);
    const res = await sendRequired<typeof setReqZ, typeof setResZ>(
      this.client,
      SET_ENDPOINT,
      {
        statuses: array.toArray(statuses),
        parent: opts.parent,
      },
      setReqZ,
      setResZ,
    );
    return isMany ? res.statuses : res.statuses[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
