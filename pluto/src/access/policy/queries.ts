// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, actions, resource } from "@synnaxlabs/client";
import { array, deep, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "policy";

export const formSchema = z.object({
  key: z.uuid().optional(),
  name: z.string().min(1, "Name is required"),
  objects: resource.idZ.array(),
  actions: access.actionZ.array(),
  role: access.role.keyZ,
});

export const formPath = deep.path<z.infer<typeof formSchema>>();

export type RetrieveQuery = {
  key: access.policy.Key;
};

export const { use } = Flux.createRetrieve<RetrieveQuery, access.policy.Policy>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.policies.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.access.policies.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.access.policies.watch(query).getCached(),
});

export type DeleteParams = access.policy.Key | access.policy.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    await client.access.policies.delete(keys, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams {
  key: access.policy.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.access.policies.rename(key, name);
    return data;
  },
});

export const useForm = Flux.createForm<RetrieveQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: {
    key: undefined,
    name: "",
    objects: [],
    actions: [],
    role: "",
  },
  retrieve: async ({ client, query: { key } }) =>
    await client.access.policies.retrieve(key),
  update: async ({ client, value, set, stored }) => {
    const values = value();
    if (values.key == null) {
      const created = await client.access.policies.create(values);
      set(formPath.key, created.key);
      return;
    }
    const edits = actions.diffFields(formSchema, stored, values, [
      "name",
      "objects",
      "actions",
      "role",
    ]);
    if (edits.length > 0) await client.access.policies.dispatch(values.key, edits);
  },
});
