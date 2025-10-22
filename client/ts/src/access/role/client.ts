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
import { z } from "zod";

import { keyZ, type NewRole, newRoleZ, type Role, roleZ } from "@/access/role/payload";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
});

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const singleCreateArgsZ = newRoleZ.transform((r) => ({ roles: [r] }));
export type SingleCreateArgs = z.input<typeof singleCreateArgsZ>;

export const multipleCreateArgsZ = newRoleZ.array().transform((roles) => ({ roles }));

export const createArgsZ = z.union([singleCreateArgsZ, multipleCreateArgsZ]);
export type CreateArgs = z.input<typeof createArgsZ>;

const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: array.nullableZ(roleZ) });

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

export const retrieveArgsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const deleteResZ = z.object({});

const deleteArgsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteArgs = z.input<typeof deleteArgsZ>;

const RETRIEVE_ENDPOINT = "/access/role/retrieve";
const CREATE_ENDPOINT = "/access/role/create";
const DELETE_ENDPOINT = "/access/role/delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(role: NewRole): Promise<Role>;
  async create(roles: NewRole[]): Promise<Role[]>;
  async create(roles: NewRole | NewRole[]): Promise<Role | Role[]> {
    const isMany = Array.isArray(roles);
    const res = await sendRequired<typeof createArgsZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
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
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    return isSingle ? res.roles[0] : res.roles;
  }

  async delete(args: DeleteArgs): Promise<void> {
    await sendRequired<typeof deleteArgsZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      args,
      deleteArgsZ,
      deleteResZ,
    );
  }
}
