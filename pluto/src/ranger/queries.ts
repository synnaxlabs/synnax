// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ranger } from "@synnaxlabs/client";
import { z } from "zod/v4";

import { Ontology } from "@/ontology";
import { Query } from "@/query";
import { parsedHandler } from "@/query/listeners";
import { type UseReturn } from "@/query/query";
import { Synnax } from "@/synnax";

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Query.useParsedListener(ranger.SET_CHANNEL_NAME, ranger.payloadZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void =>
  Query.useParsedListener(ranger.DELETE_CHANNEL_NAME, ranger.keyZ, onDelete);

export const useAliasSetSynchronizer = (onSet: (alias: ranger.Alias) => void): void =>
  Query.useParsedListener(ranger.SET_ALIAS_CHANNEL_NAME, ranger.aliasZ, onSet);

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void =>
  Query.useStringListener(
    ranger.DELETE_ALIAS_CHANNEL_NAME,
    ranger.decodeDeleteAliasChange,
    onDelete,
  );

export const useChildren = (key: ranger.Key): UseReturn<ranger.Range[]> => {
  const res = Ontology.useChildren(ranger.ontologyID(key));
  const client = Synnax.use();
  if (res.status !== "success") return res;
  if (client == null) return { ...res, data: [] };
  return {
    ...res,
    data: res.data
      .filter(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE)
      .map((child) => client.ranges.sugarOntologyResource(child)),
  };
};

export const useParent = (key: ranger.Key): UseReturn<ranger.Range | null> => {
  const res = Ontology.useParents(ranger.ontologyID(key));
  const client = Synnax.use();
  if (res.status !== "success") return res;
  if (client == null) return { ...res, data: null };
  const parent = res.data.find(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE);
  if (parent == null) return { ...res, data: null };
  return { ...res, data: client.ranges.sugarOntologyResource(parent) };
};

const SET_LISTENER_CONFIG: Query.ListenerConfig<ranger.Key, ranger.Range> = {
  channel: ranger.SET_CHANNEL_NAME,
  onChange: parsedHandler(
    ranger.payloadZ,
    async ({ client, changed, params: key, onChange }) => {
      if (changed.key !== key) return;
      onChange(client.ranges.sugarOne(changed));
    },
  ),
};

export const use = Query.create<ranger.Key, ranger.Range>({
  name: "Range",
  queryFn: async ({ client, params: key }) => client.ranges.retrieve(key),
  listeners: [SET_LISTENER_CONFIG],
});

export const rangeFormSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: z.object({ start: z.number(), end: z.number() }),
});

export const useForm = Query.createForm<ranger.Key, typeof rangeFormSchema>({
  name: "Range",
  schema: rangeFormSchema,
  queryFn: async ({ client, params: key }) => {
    if (key == null) return null;
    const range = await client.ranges.retrieve(key);
    return {
      ...range.payload,
      timeRange: range.timeRange.numeric,
      labels: (await range.labels()).map((l) => l.key),
    };
  },
  mutationFn: async ({ client, values }) => {
    const rng = await client.ranges.create(values);
    await client.labels.label(rng.key, values.labels, { replace: true });
    return rng;
  },
  listeners: [],
});
