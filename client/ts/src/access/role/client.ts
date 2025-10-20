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

import {
  type Key,
  keyZ,
  type NewRole,
  newRoleZ,
  type Role,
  roleZ,
} from "@/access/role/payload";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
});

const createReqZ = z.object({ roles: newRoleZ.array() });
const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: array.nullableZ(roleZ) });
const updateReqZ = z.object({ role: roleZ });
const updateResZ = z.object({});
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const RETRIEVE_ENDPOINT = "/access/role/retrieve";
const CREATE_ENDPOINT = "/access/role/create";
const UPDATE_ENDPOINT = "/access/role/update";
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
    const parsedRoles = newRoleZ.array().parse(array.toArray(roles));
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      createReqZ,
      createResZ,
    )({ roles: parsedRoles });
    return isMany ? res.roles : res.roles[0];
  }

  async retrieve(key: Key): Promise<Role>;
  async retrieve(keys: Key[]): Promise<Role[]>;
  async retrieve(): Promise<Role[]>;
  async retrieve(keys?: Key | Key[]): Promise<Role | Role[]> {
    const parsedKeys = keys != null ? array.toArray(keys) : undefined;
    const isSingle = typeof keys === "string";
    const req = parsedKeys != null ? { keys: parsedKeys } : {};
    const res = await sendRequired<typeof retrieveRequestZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      retrieveRequestZ,
      retrieveResZ,
    )(req);
    return isSingle ? res.roles[0] : res.roles;
  }

  async update(role: Role): Promise<void> {
    await sendRequired<typeof updateReqZ, typeof updateResZ>(
      this.client,
      UPDATE_ENDPOINT,
      updateReqZ,
      updateResZ,
    )({ role });
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    const parsedKeys = array.toArray(keys);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      deleteReqZ,
      deleteResZ,
    )({ keys: parsedKeys });
  }
}
