// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Key, keyZ, type New, type Role, roleZ } from "@/access/role/types.gen";
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

const singleCreateParamsZ = roleZ.transform((r) => ({ roles: [r] }));
export type SingleCreateParams = z.input<typeof singleCreateParamsZ>;

export const multipleCreateParamsZ = roleZ.array().transform((roles) => ({ roles }));

export const createParamsZ = z.union([singleCreateParamsZ, multipleCreateParamsZ]);
export type CreateParams = z.input<typeof createParamsZ>;

const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: roleZ.array().default(() => []) });

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

export const retrieveParamsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const deleteResZ = z.object({});

const deleteParamsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteParams = z.input<typeof deleteParamsZ>;

const assignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type AssignParams = z.input<typeof assignReqZ>;

const assignResZ = z.object({});

const unassignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type UnassignParams = z.input<typeof unassignReqZ>;

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
    const res = await this.client.send(
      "/access/role/create",
      roles,
      createParamsZ,
      createResZ,
    );
    return isMany ? res.roles : res.roles[0];
  }

  async retrieve(key: Key): Promise<Role>;
  async retrieve(params: RetrieveSingleParams): Promise<Role>;
  async retrieve(params: RetrieveMultipleParams): Promise<Role[]>;
  async retrieve(params: Key | RetrieveParams): Promise<Role | Role[]> {
    if (typeof params !== "object") params = { key: params };
    const isSingle = "key" in params;
    const res = await this.client.send(
      "/access/role/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return isSingle ? res.roles[0] : res.roles;
  }

  async delete(params: DeleteParams): Promise<void> {
    await this.client.send("/access/role/delete", params, deleteParamsZ, deleteResZ);
  }

  async assign(params: AssignParams): Promise<void> {
    await this.client.send("/access/role/assign", params, assignReqZ, assignResZ);
  }

  async unassign(params: UnassignParams): Promise<void> {
    await this.client.send("/access/role/unassign", params, unassignReqZ, unassignResZ);
  }
}
