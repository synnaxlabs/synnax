// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface ListQuery {
  /** Limits the list to resources of these types. Empty lists every type. */
  types?: resource.Type[];
}

/**
 * Lists the records of every type that match the search term, closest match first. An
 * empty term lists the records in order of type. The list stays current while it is
 * mounted.
 */
export const useList = (
  params: Omit<
    Flux.UseListParams<ListQuery, resource.Resource>,
    "itemQuery" | "name"
  > = {},
): Flux.UseListReturn<string, resource.Resource> =>
  Flux.useList((client) => client.resources, {
    ...params,
    name: "resources",
    itemQuery: resource.parseID,
  });
