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
import { array, uuid, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

export type UseDeleteParams = user.Key | user.Key[];

const RESOURCE_NAME = "user";
const PASSWORD_RESOURCE_NAME = "password";

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

export interface ChangeUsernameParams extends Pick<user.User, "key" | "username"> {}

export const { useUpdate: useRename } = Flux.createUpdate<ChangeUsernameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, username } = data;
    await client.users.changeUsername(key, username, {
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
  password: z.string().min(1, "Password is required"),
  firstName: z.string().min(1, "First name is required"),
  lastName: z.string().min(1, "Last name is required"),
  role: access.role.keyZ,
});

export type FormQuery = {
  key: user.Key;
};

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  username: "",
  firstName: "",
  lastName: "",
  password: "",
  role: "",
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: "user",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key } }) => ({
    ...(await client.users.retrieve(key)),
    password: "",
    role: "",
  }),
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

const passwordFieldsZ = z.object({
  password: z.string().min(1, "Password is required"),
  confirmPassword: z.string(),
});

const passwordsMatch = ({
  password,
  confirmPassword,
}: z.infer<typeof passwordFieldsZ>): boolean => password === confirmPassword;

const MISMATCH_ISSUE = {
  error: "Passwords do not match",
  path: ["confirmPassword"],
};

export const changePasswordFormSchema = passwordFieldsZ
  .extend({ key: user.keyZ })
  .refine(passwordsMatch, MISMATCH_ISSUE);

export type ChangePasswordFormQuery = Record<string, never>;

const ZERO_CHANGE_PASSWORD_FORM_VALUES: z.infer<typeof changePasswordFormSchema> = {
  key: "",
  password: "",
  confirmPassword: "",
};

/**
 * Sets another user's password. The subject needs update access on that user; the
 * user's current password is not required. Pass the target key through
 * initialValues.
 */
export const useChangePasswordForm = Flux.createForm<
  ChangePasswordFormQuery,
  typeof changePasswordFormSchema
>({
  name: PASSWORD_RESOURCE_NAME,
  schema: changePasswordFormSchema,
  initialValues: ZERO_CHANGE_PASSWORD_FORM_VALUES,
  update: async ({ client, value }) => {
    const { key, password } = value();
    await client.users.changePassword(key, password);
  },
});

export const changeOwnPasswordFormSchema = passwordFieldsZ
  .extend({
    currentPassword: z.string().min(1, "Current password is required"),
  })
  .refine(passwordsMatch, MISMATCH_ISSUE);

export type ChangeOwnPasswordFormQuery = Record<string, never>;

const ZERO_CHANGE_OWN_PASSWORD_FORM_VALUES: z.infer<
  typeof changeOwnPasswordFormSchema
> = { currentPassword: "", password: "", confirmPassword: "" };

/**
 * Sets the signed-in user's own password. The user types their current password
 * rather than the client replaying the one it holds, so an unattended session cannot
 * change its own password.
 */
export const useChangeOwnPasswordForm = Flux.createForm<
  ChangeOwnPasswordFormQuery,
  typeof changeOwnPasswordFormSchema
>({
  name: PASSWORD_RESOURCE_NAME,
  schema: changeOwnPasswordFormSchema,
  initialValues: ZERO_CHANGE_OWN_PASSWORD_FORM_VALUES,
  update: async ({ client, value }) => {
    const { currentPassword, password } = value();
    await client.auth.changePassword(currentPassword, password);
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

export const { use, useResult, createResultSelector } = Flux.createRetrieve<
  Partial<RetrieveQuery>,
  user.User
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => {
    if (key == null) return await retrieveCurrent(client);
    return await client.users.retrieve(key);
  },
  onChange: ({ client, query: { key } }, handler) => {
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

export const useResultKey = createResultSelector(({ key }) => key);

export const useResultUsername = createResultSelector(({ username }) => username);

export const useResultFirstName = createResultSelector(({ firstName }) => firstName);

export const useResultLastName = createResultSelector(({ lastName }) => lastName);
