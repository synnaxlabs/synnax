// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AccountLinkedError,
  type auth,
  ProviderLinkedError,
  type resource,
  status,
} from "@synnaxlabs/client";
import { errors, verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export type RetrieveQuery = {
  subject: resource.ID;
};

export const { useResult, useInvalidate, useRefresh } = Flux.createRetrieve<
  RetrieveQuery,
  auth.oidc.Link[]
>({
  name: "provider accounts",
  retrieve: async ({ client, query: { subject } }) =>
    await client.auth.oidc.retrieve([subject]),
});

// The Core's reason names no provider, so name the one the person chose.
const describeRefusal = (e: unknown, provider: string): string | undefined => {
  if (ProviderLinkedError.matches(e))
    return `You already have a ${provider} account linked.`;
  if (AccountLinkedError.matches(e))
    return `This ${provider} account is linked to another user.`;
  return undefined;
};

const toLinkError = (e: unknown, provider: string): Error => {
  const description = describeRefusal(e, provider);
  if (description == null) return errors.fromUnknown(e);
  return status.toError(
    status.create({ variant: "error", message: "Failed to link account", description }),
  );
};

export const { useUpdate: useLink } = Flux.createUpdate<
  auth.OIDCCredentials,
  auth.oidc.Link
>({
  name: "account",
  verbs: verbs.LINK,
  update: async ({ client, data }) => {
    try {
      return await client.auth.oidc.link(data);
    } catch (e) {
      throw toLinkError(e, data.provider);
    }
  },
});

export const { useUpdate: useUnlink } = Flux.createUpdate<auth.oidc.LinkKey>({
  name: "account",
  verbs: verbs.UNLINK,
  update: async ({ client, data }) => {
    await client.auth.oidc.unlink(data);
    return data;
  },
});
