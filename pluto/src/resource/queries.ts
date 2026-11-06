// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";

import { Flux } from "@/flux";

const PLURAL_RESOURCE_NAME = "resources";

export type ListQuery = List.PagerParams & {
  /** Limits the list to resources of these types. Empty lists every type. */
  types?: resource.Type[];
};

/**
 * Lists the records of every type that match the search term, closest match first. An
 * empty term lists the records in order of type. The list stays current while it is
 * mounted.
 */
export const useList = Flux.createList<ListQuery, string, resource.Resource>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.resources.search(query),
  retrieveByKey: async ({ client, key }) =>
    await client.resources.retrieve(resource.parseID(key)),
  onChange: ({ client, query }, handler) => client.resources.onChange(query, handler),
  getCached: ({ client, query }) => client.resources.getCached(query),
  onChangeByKey: ({ client, key }, handler) =>
    client.resources.onChange(resource.parseID(key), handler),
});
