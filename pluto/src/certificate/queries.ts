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

/** Lists certificates by name, unless the query sets an order. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, certificate.Certificate>,
  "name"
> = {}): Flux.UseListReturn<certificate.Key, certificate.Certificate> =>
  Flux.useList((client) => client.certificates, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : { order: { field: "name" }, ...query },
  });

export type RetrieveQuery = certificate.RetrieveSingleParams;

export const { use } = Flux.createRetrieve<RetrieveQuery, certificate.Certificate>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.certificates.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.certificates.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.certificates.watch(key).getCached(),
});

export interface RenameParams extends Pick<certificate.Certificate, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.certificates.rename(key, name);
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
