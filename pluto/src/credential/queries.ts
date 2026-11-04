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

export const { useResult: useResultAPIKeys, useInvalidate: useInvalidateAPIKeys } =
  Flux.createRetrieve<SubjectQuery, auth.apiKey.APIKey[]>({
    name: "API keys",
    retrieve: async ({ client, query: { subject } }) =>
      await client.auth.apiKeys.retrieve([subject]),
  });

export const { useResult: useResultSessions, useInvalidate: useInvalidateSessions } =
  Flux.createRetrieve<SubjectQuery, auth.Session[]>({
    name: "sessions",
    retrieve: async ({ client, query: { subject } }) =>
      await client.auth.sessions.retrieve([subject]),
  });

export interface CreateAPIKeyParams {
  subject: ontology.ID;
  name: string;
}

export interface CreateAPIKeyResult {
  /** The only copy of the key's secret. */
  token: string;
  key: auth.apiKey.APIKey;
}

export const { useUpdate: useCreateAPIKey } = Flux.createUpdate<
  CreateAPIKeyParams,
  CreateAPIKeyResult
>({
  name: "API key",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.auth.apiKeys.create(data),
});

export const { useUpdate: useDeleteAPIKey } = Flux.createUpdate<auth.apiKey.Key>({
  name: "API key",
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.auth.apiKeys.delete(data);
    return data;
  },
});

export const { useUpdate: useSetPassword } = Flux.createUpdate<auth.password.SetParams>(
  {
    name: "password",
    verbs: verbs.SET,
    update: async ({ client, data }) => {
      await client.auth.passwords.set(data);
      return data;
    },
  },
);
