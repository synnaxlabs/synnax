// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";

import { Ontology } from "@/ontology";
import { Sync } from "@/sync";
import { Synnax } from "@/synnax";

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Sync.useParsedListener(ranger.SET_CHANNEL_NAME, ranger.payloadZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void =>
  Sync.useParsedListener(ranger.DELETE_CHANNEL_NAME, ranger.keyZ, onDelete);

export const useAliasSetSynchronizer = (onSet: (alias: ranger.Alias) => void): void =>
  Sync.useParsedListener(ranger.SET_ALIAS_CHANNEL_NAME, ranger.aliasZ, onSet);

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void =>
  Sync.useStringListener(
    ranger.DELETE_ALIAS_CHANNEL_NAME,
    ranger.decodeDeleteAliasChange,
    onDelete,
  );

export const useChildren = (key: ranger.Key): ranger.Range[] => {
  const children = Ontology.useChildren(ranger.ontologyID(key)).filter(
    ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
  );
  const client = Synnax.use();
  if (client == null) return [];
  return children.map((child) => client.ranges.sugarOntologyResource(child));
};

export const useParent = (key: ranger.Key): ranger.Range | null => {
  const parent = Ontology.useParents(ranger.ontologyID(key)).find(
    ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
  );
  const client = Synnax.use();
  if (parent == null || client == null) return null;
  return client.ranges.sugarOntologyResource(parent);
};

export const use = Sync.createQuery<ranger.Key, ranger.Range>({
  queryFn: async (client, key) => client.ranges.retrieve(key),
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: async (client, series, key) => {
        const parsed = series.parseJSON(ranger.payloadZ);
        const found = parsed.find((p) => p.key === key);
        if (found == null) return null;
        return client.ranges.sugarOne(found);
      },
    },
  ],
});
