// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ranger } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

export const RESOURCE_NAME = "channel";
export const PLURAL_RESOURCE_NAME = "channels";

export type RetrieveQuery = {
  key: channel.Key;
  rangeKey?: ranger.Key;
};

export const retrieveDefinition: flux.Definition<RetrieveQuery, channel.Channel> = {
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key, rangeKey } }) =>
    await client.channels.retrieve(key, { rangeKey }),
  onChange: ({ client, query }, handler) => client.channels.onChange(query, handler),
  getCached: ({ client, query }) => client.channels.getCached(query),
};

export type RetrieveMultipleQuery = channel.RetrieveOptions & {
  keys: channel.Key[];
};

export const retrieveMultipleDefinition: flux.Definition<
  RetrieveMultipleQuery,
  channel.Channel[]
> = {
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.channels.retrieve(query),
  onChange: ({ client, query }, handler) => client.channels.onChange(query, handler),
  getCached: ({ client, query }) => client.channels.getCached(query),
};
