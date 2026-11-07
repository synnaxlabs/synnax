// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

const RESOURCE_NAME = "certificate";
const PLURAL_RESOURCE_NAME = "certificates";

export type ListQuery = certificate.RetrieveMultipleParams;

export const useList = Flux.createList<
  ListQuery,
  certificate.Key,
  certificate.Certificate
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.certificates.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.certificates.retrieve(key),
  onChange: ({ client, query }, handler) =>
    client.certificates.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) =>
    client.certificates.onChange(key, handler),
  getCached: ({ client, query }) => client.certificates.getCached(query),
});

export type RetrieveQuery = certificate.RetrieveSingleParams;

export const { use } = Flux.createRetrieve<RetrieveQuery, certificate.Certificate>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.certificates.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.certificates.onChange(key, handler),
  getCached: ({ client, query: { key } }) => client.certificates.getCached(key),
});

export interface RenameParams extends Pick<certificate.Certificate, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.certificates.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = certificate.Key | certificate.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.certificates.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
