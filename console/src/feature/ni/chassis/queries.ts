// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { Flux } from "@synnaxlabs/pluto";
import { verbs } from "@synnaxlabs/x";

const RESOURCE_NAME = "NI chassis";

const BASE_QUERY = { includeStatus: true } as const;

export type RetrieveQuery = ni.chassis.RetrieveSingleParams;

/** The NI chassis with the key, with its status. */
export const { useResult: useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  ni.chassis.Chassis
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.ni.chassis.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.ni.chassis.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.ni.chassis.watch(query).getCached(),
});

export type ListQuery = ni.chassis.RetrieveRequest;

/** Lists NI chassis with their statuses. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, ni.chassis.Chassis>,
  "name"
> = {}): Flux.UseListReturn<ni.chassis.Key, ni.chassis.Chassis> =>
  Flux.useList((client) => client.ni.chassis, {
    ...rest,
    name: RESOURCE_NAME,
    query: query === null ? null : { ...BASE_QUERY, ...query },
  });

export interface RenameParams extends Pick<ni.chassis.Chassis, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.ni.chassis.update(key, { group: "configuration", name });
    return data;
  },
});

export type DeleteParams = ni.chassis.Key | ni.chassis.Key[];

/** Deletes NI chassis and the devices they hold. */
export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.ni.chassis.delete(data);
    return data;
  },
});
