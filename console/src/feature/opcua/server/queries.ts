// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { opcua } from "@synnaxlabs/client";
import { Flux } from "@synnaxlabs/pluto";
import { verbs } from "@synnaxlabs/x";

const RESOURCE_NAME = "OPC UA server";

const BASE_QUERY = { includeStatus: true } as const;

export type RetrieveQuery = opcua.server.RetrieveSingleParams;

/** The OPC UA server with the key, with its status, from the typed server table. */
export const { useResult: useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  opcua.server.Server
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.opcua.servers.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.opcua.servers.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.opcua.servers.watch(query).getCached(),
});

export interface RenameParams extends Pick<opcua.server.Server, "key" | "name"> {}

/**
 * Renames an OPC UA server. The Core clears each identity field that an update leaves
 * out, so the rename sends the stored identity with the new name.
 */
export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    const stored = opcua.server.identityZ.parse(
      await client.opcua.servers.retrieve({ key }),
    );
    await client.opcua.servers.update(key, { ...stored, name, group: "identity" });
    return data;
  },
});

export type DeleteParams = opcua.server.Key | opcua.server.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.opcua.servers.delete(data);
    return data;
  },
});
