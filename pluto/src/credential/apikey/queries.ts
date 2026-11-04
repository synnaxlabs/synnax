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

export type RetrieveQuery = {
  subject: ontology.ID;
};

export const { useResult, useInvalidate } = Flux.createRetrieve<
  RetrieveQuery,
  auth.apiKey.APIKey[]
>({
  name: "API keys",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.apiKeys.retrieve([subject]),
});

export interface CreateParams {
  subject: ontology.ID;
  name: string;
}

export interface CreateResult {
  /** The only copy of the key's secret. */
  token: string;
  key: auth.apiKey.APIKey;
}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, CreateResult>({
  name: "API key",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.auth.apiKeys.create(data),
});

export const { useUpdate: useDelete } = Flux.createUpdate<auth.apiKey.Key>({
  name: "API key",
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.auth.apiKeys.delete(data);
    return data;
  },
});
