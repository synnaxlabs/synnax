// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, type ontology } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export type SubjectQuery = {
  subject: ontology.ID;
};

export const { useResult: useResultKeys, useInvalidate: useInvalidateKeys } =
  Flux.createRetrieve<SubjectQuery, auth.APIKey[]>({
    name: "API keys",
    retrieve: async ({ client, query: { subject } }) =>
      await client.auth.retrieveKeys([subject]),
  });

export const { useResult: useResultSessions, useInvalidate: useInvalidateSessions } =
  Flux.createRetrieve<SubjectQuery, auth.Session[]>({
    name: "sessions",
    retrieve: async ({ client, query: { subject } }) =>
      await client.auth.retrieveSessions([subject]),
  });

export interface CreateKeyParams {
  subject: ontology.ID;
  name: string;
}

export interface CreateKeyResult {
  /** The only copy of the key's secret. */
  token: string;
  key: auth.APIKey;
}

export const { useUpdate: useCreateKey } = Flux.createUpdate<
  CreateKeyParams,
  CreateKeyResult
>({
  name: "API key",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.auth.createKey(data),
});

export const { useUpdate: useDeleteKey } = Flux.createUpdate<auth.APIKeyKey>({
  name: "API key",
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.auth.deleteKeys(data);
    return data;
  },
});

export const { useUpdate: useSetPassword } = Flux.createUpdate<auth.SetPasswordParams>({
  name: "password",
  verbs: verbs.SET,
  update: async ({ client, data }) => {
    await client.auth.setPassword(data);
    return data;
  },
});
