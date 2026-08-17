// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

const PLURAL_RESOURCE_NAME = "ranges";

export type ListQuery = Omit<ranger.RetrieveRequest, "names">;

export const listDefinition: flux.Definition<ListQuery, ranger.Range[]> = {
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.ranges.retrieve(query),
  onChange: ({ client, query }, handler) => client.ranges.onChange(query, handler),
  getCached: ({ client, query }) => client.ranges.getCached(query),
};
