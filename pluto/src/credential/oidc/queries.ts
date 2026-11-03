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
  auth.oidc.Link[]
>({
  name: "provider accounts",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.oidc.retrieve([subject]),
});

export const { useUpdate: useLink } = Flux.createUpdate<
  auth.OIDCCredentials,
  auth.oidc.Link
>({
  name: "provider account",
  verbs: verbs.LINK,
  update: async ({ client, data }) => await client.auth.oidc.link(data),
});

export const { useUpdate: useUnlink } = Flux.createUpdate<auth.oidc.LinkKey>({
  name: "provider account",
  verbs: verbs.UNLINK,
  update: async ({ client, data }) => {
    await client.auth.oidc.unlink(data);
    return data;
  },
});
