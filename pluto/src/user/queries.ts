// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology, user } from "@synnaxlabs/client";
import { array, uuid, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

export type UseDeleteParams = user.Key | user.Key[];

const RESOURCE_NAME = "user";

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    await client.users.delete(keys, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type RetrieveQuery = {
  key: user.Key;
};

export interface RenameParams extends Pick<user.User, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.users.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type UseRetrieveGroupParams = Record<string, never>;

export const { use: useGroupID } = Flux.createRetrieve<
  UseRetrieveGroupParams,
  ontology.ID | undefined
>({
  name: "user group",
  retrieve: async ({ client }) => {
    const res = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
    return res.find((r) => r.name === "Users")?.id;
  },
});

export const formSchema = user.newZ.extend({
  username: z.string().min(1, "username is required"),
  password: z.string().min(1, "Password is required"),
  name: z.string().min(1, "Name is required"),
  role: access.role.keyZ,
});

export type FormQuery = {
  key: user.Key;
};

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  username: "",
  name: "",
  password: "",
  role: "",
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: "user",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  update: async ({ client, value }) => {
    const v = value();
    const createdUser = await client.users.create({ key: uuid.create(), ...v });
    if (v.role == null) return;
    await client.access.roles.assign({
      user: createdUser.key,
      role: v.role,
    });
  },
});

export const { use, useResult, createResultSelector } = Flux.createRetrieve<
  Partial<RetrieveQuery>,
  user.User
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    key == null
      ? await client.users.current.retrieve()
      : await client.users.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    key == null
      ? client.users.current.onChange(handler)
      : client.users.onChange(key, handler),
  getCached: ({ client, query: { key } }) =>
    key == null ? client.users.current.getCached() : client.users.getCached(key),
});

export const useResultKey = createResultSelector(({ key }) => key);

export const useResultName = createResultSelector(({ name }) => name);
