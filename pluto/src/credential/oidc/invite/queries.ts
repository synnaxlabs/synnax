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
  auth.oidc.invite.Invite[]
>({
  name: "invites",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.oidc.invites.retrieve(subject),
});

export interface CreateResult {
  /** The only copy of the invite's secret. */
  token: string;
  invite: auth.oidc.invite.Invite;
}

export const { useUpdate: useCreate } = Flux.createUpdate<ontology.ID, CreateResult>({
  name: "invite",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.auth.oidc.invites.create(data),
});

export const { useUpdate: useRevoke } = Flux.createUpdate<auth.oidc.invite.Key>({
  name: "invite",
  verbs: verbs.REVOKE,
  update: async ({ client, data }) => {
    await client.auth.oidc.invites.delete(data);
    return data;
  },
});
