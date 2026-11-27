// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth, type resource } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export type RetrieveQuery = {
  subject: resource.ID;
};

export const { useResult, useInvalidate, useRefresh } = Flux.createRetrieve<
  RetrieveQuery,
  auth.password.Password[]
>({
  name: "password",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.passwords.retrieve({ subjects: [subject] }),
});

export const { useUpdate: useCreate } = Flux.createUpdate<auth.password.CreateParams>({
  name: "password",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    await client.auth.passwords.create(data);
    return data;
  },
});

export interface RenameParams {
  /** The password to rename. */
  key: auth.password.Key;
  /** The new username. */
  username: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: "password",
  verbs: verbs.RENAME,
  update: async ({ client, data: { key, username } }) => {
    await client.auth.passwords.dispatch(key, auth.password.setUsername({ username }));
    return { key, username };
  },
});

export const { useUpdate: useSet } = Flux.createUpdate<auth.password.SetParams>({
  name: "password",
  verbs: verbs.SET,
  update: async ({ client, data }) => {
    await client.auth.passwords.set(data);
    return data;
  },
});
