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

export type SearchQuery = List.PagerParams & {
  /** Limits the search to resources of these types. Empty searches every type. */
  types?: resource.Type[];
};

/**
 * Searches the records of every type for the search term, closest match first. An
 * empty term matches nothing.
 */
export const useSearch = Flux.createList<SearchQuery, string, resource.Resource>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { searchTerm = "", types, limit, offset } }) =>
    await client.resources.search({ term: searchTerm, types, limit, offset }),
  retrieveByKey: async ({ client, key }) =>
    await client.resources.retrieve(resource.parseID(key)),
});
