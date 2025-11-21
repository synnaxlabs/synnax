// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import z from "zod";

import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";

const roleFormSchema = z.object({
  key: z.uuid().optional(),
  name: z.string(),
  description: z.string().optional(),
});

export interface RetrieveRoleQuery {
  key: string;
}

const retrieveSingleRole = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<
  RetrieveRoleQuery,
  role.FluxSubStore
>): Promise<access.role.Role> => {
  let r = store.roles.get(key);
  if (r != null) return r;
  r = await client.access.roles.retrieve({ key });
  store.roles.set(key, r);
  return r;
};

export const useRoleForm = Flux.createForm<
  Partial<RetrieveRoleQuery>,
  typeof roleFormSchema,
  role.FluxSubStore
>({
  name: "Role",
  schema: roleFormSchema,
  initialValues: {
    key: undefined,
    name: "",
    description: "",
  },
  retrieve: async ({ client, query, store }) => {
    if (query.key == null) return;
    const role = await retrieveSingleRole({ client, query: { key: query.key }, store });
    store.roles.set(query.key, role);
  },
  update: async ({ client, value, store, set, rollbacks }) => {
    const v = value();
    let r: access.role.Role = { key: uuid.create(), ...v };
    rollbacks.push(store.roles.set(r.key, r));
    r = await client.access.roles.create(r);
    set("key", r.key);
  },
});
