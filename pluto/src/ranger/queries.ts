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
import { type UseReturn } from "@/query/query";
import { Sync } from "@/query/sync";
import { Synnax } from "@/synnax";

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Sync.useListener({
    channel: ranger.SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ranger.payloadZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void =>
  Sync.useListener({
    channel: ranger.DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ranger.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });

export const useAliasSetSynchronizer = (onSet: (alias: ranger.Alias) => void): void =>
  Sync.useListener({
    channel: ranger.SET_ALIAS_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ranger.aliasZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void =>
  Sync.useListener({
    channel: ranger.DELETE_ALIAS_CHANNEL_NAME,
    onChange: Sync.stringHandler(async (args) => {
      onDelete(ranger.decodeDeleteAliasChange(args.changed));
    }),
  });

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
  onChange: Sync.parsedHandler(
    ranger.payloadZ,
    async ({ client, changed, params: key, onChange }) => {
      if (changed.key !== key) return;
      onChange(client.ranges.sugarOne(changed));
    },
  ),
};

export const use = (key: ranger.Key) =>
  Query.use({
    name: "Range",
    params: key,
    retrieve: async ({ client, params: key }) => await client.ranges.retrieve(key),
    listeners: [SET_LISTENER_CONFIG],
  });

export const rangeFormSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: z.object({ start: z.number(), end: z.number() }),
});

export const rangeToFormValues = async (
  range: ranger.Range,
  labels?: label.Key[],
  parent?: ranger.Key,
) => ({
  ...range.payload,
  timeRange: range.timeRange.numeric,
  labels: labels ?? (await range.labels()).map((l) => l.key),
  parent: parent ?? (await range.retrieveParent())?.key,
});

export const useForm = (
  args: Pick<
    Query.UseFormArgs<ranger.Key, typeof rangeFormSchema>,
    "initialValues" | "params"
  >,
) =>
  Query.useForm<ranger.Key, typeof rangeFormSchema>({
    ...args,
    name: "Range",
    schema: rangeFormSchema,
    retrieve: async ({ client, params: key }) => {
      if (key == null) return null;
      const rng = await client.ranges.retrieve(key);
      return await rangeToFormValues(rng);
    },
    update: async ({ client, values }) => {
      const rng = await client.ranges.create(values);
      await client.labels.label(rng.key, values.labels, { replace: true });
      return await rangeToFormValues(rng, values.labels, values.parent);
    },
    listeners: [
      {
        channel: ranger.SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ranger.payloadZ,
          async ({ client, changed, params: key, onChange }) => {
            if (changed.key !== key) return;
            onChange(await rangeToFormValues(client.ranges.sugarOne(changed)));
          },
        ),
      },
    ],
  });
