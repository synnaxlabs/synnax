// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  ontology,
  type Synnax,
  UnexpectedError,
  user,
} from "@synnaxlabs/client";
import { array, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

export type UseDeleteParams = user.Key | user.Key[];

const RESOURCE_NAME = "user";

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
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

export interface ChangeUsernameParams extends Pick<user.User, "key" | "username"> {}

export const { useUpdate: useRename } = Flux.createUpdate<ChangeUsernameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data }) => {
    const { key, username } = data;
    await client.users.changeUsername(key, username);
    return data;
  },
});

export type UseRetrieveGroupParams = Record<string, never>;

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupParams,
  ontology.ID | undefined
>({
  name: "User Group",
  retrieve: async ({ client }) => {
    const res = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
    return res.find((r) => r.name === "Users")?.id;
  },
});

export const formSchema = user.newZ.extend({
  password: z.string().min(1, "Password is required"),
  firstName: z.string().min(1, "First name is required"),
  lastName: z.string().min(1, "Last name is required"),
  role: access.role.keyZ,
});

export type UseFormParams = {
  key?: user.Key;
};

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  username: "",
  firstName: "",
  lastName: "",
  password: "",
  role: "",
};

export const useForm = Flux.createForm<UseFormParams, typeof formSchema>({
  name: "User",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    const user = await client.users.retrieve(key);
    reset({ ...user, password: "", role: "" });
  },
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

const retrieveCurrent = async (client: Synnax): Promise<user.User> => {
  const user = client.auth?.user;
  if (user == null) {
    await client.connect();
    if (client.auth?.user == null)
      throw new UnexpectedError(
        "Expected user to be available after successfully connecting to cluster",
      );
    return client.auth.user;
  }
  return user;
};

export const { useRetrieve } = Flux.createRetrieve<Partial<RetrieveQuery>, user.User>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => {
    if (key == null) return await retrieveCurrent(client);
    return await client.users.retrieve(key);
  },
  subscribe: ({ client, query: { key } }, handler) => {
    key ??= client.auth?.user?.key;
    if (key == null) return () => {};
    return client.users.onChange(key, handler);
  },
  getCached: ({ client, query: { key } }) => {
    key ??= client.auth?.user?.key;
    if (key == null) return undefined;
    return client.users.getCached(key);
  },
});
