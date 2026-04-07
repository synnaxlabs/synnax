// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ, type New, newZ, type Role, roleZ } from "@/access/role/types.gen";
import { user } from "@/user";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const singleCreateArgsZ = newZ.transform((r) => ({ roles: [r] }));
export type SingleCreateArgs = z.input<typeof singleCreateArgsZ>;

export const multipleCreateArgsZ = newZ.array().transform((roles) => ({ roles }));

export const createArgsZ = z.union([singleCreateArgsZ, multipleCreateArgsZ]);
export type CreateArgs = z.input<typeof createArgsZ>;

const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: array.nullishToEmpty(roleZ) });

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

export const retrieveArgsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const deleteResZ = z.object({});

const deleteArgsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteArgs = z.input<typeof deleteArgsZ>;

const assignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type AssignArgs = z.input<typeof assignReqZ>;

const assignResZ = z.object({});

const unassignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type UnassignArgs = z.input<typeof unassignReqZ>;

const unassignResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_role_set";
export const DELETE_CHANNEL_NAME = "sy_role_delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(role: New): Promise<Role>;
  async create(roles: New[]): Promise<Role[]>;
  async create(roles: New | New[]): Promise<Role | Role[]> {
    const isMany = Array.isArray(roles);
    const res = await sendRequired<typeof createArgsZ, typeof createResZ>(
      this.client,
      "/access/role/create",
      roles,
      createArgsZ,
      createResZ,
    );
    return isMany ? res.roles : res.roles[0];
  }

  async retrieve(args: RetrieveSingleParams): Promise<Role>;
  async retrieve(args: RetrieveMultipleParams): Promise<Role[]>;
  async retrieve(args: RetrieveArgs): Promise<Role | Role[]> {
    const isSingle = "key" in args;
    const res = await sendRequired<typeof retrieveArgsZ, typeof retrieveResZ>(
      this.client,
      "/access/role/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    return isSingle ? res.roles[0] : res.roles;
  }

  async delete(args: DeleteArgs): Promise<void> {
    await sendRequired<typeof deleteArgsZ, typeof deleteResZ>(
      this.client,
      "/access/role/delete",
      args,
      deleteArgsZ,
      deleteResZ,
    );
  }

  async assign(args: AssignArgs): Promise<void> {
    await sendRequired<typeof assignReqZ, typeof assignResZ>(
      this.client,
      "/access/role/assign",
      args,
      assignReqZ,
      assignResZ,
    );
  }

  async unassign(args: UnassignArgs): Promise<void> {
    await sendRequired<typeof unassignReqZ, typeof unassignResZ>(
      this.client,
      "/access/role/unassign",
      args,
      unassignReqZ,
      unassignResZ,
    );
  }
}
