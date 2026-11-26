// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, control } from "@synnaxlabs/client";

import type { flux } from "@/flux/aether";

export const PLURAL_RESOURCE_NAME = "control states";

export type ListQuery = { keys: channel.Key[] };

export const listDefinition: flux.Definition<ListQuery, control.KeyedState[]> = {
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.control.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.control.watch(query.keys).onChange(handler),
  getCached: ({ client, query }) => client.control.watch(query.keys).getCached(),
};
