// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, ranger } from "@synnaxlabs/client";
import { type Optional, primitive } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Label } from "@/label";
import { Ontology } from "@/ontology";
import { Query } from "@/query";
import { Sync } from "@/query/sync";
import { type UseReturn } from "@/query/useStateful";
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
  if (res.variant !== "success") return res;
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
  if (res.variant !== "success") return res;
  const parent = res.data.find(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE);
  if (parent == null || client == null) return { ...res, data: null };
  return { ...res, data: client.ranges.sugarOntologyResource(parent) };
};

export interface QueryParams extends Query.Params {
  key: ranger.Key;
}

const SET_LISTENER_CONFIG: Query.ListenerConfig<QueryParams, ranger.Range> = {
  channel: ranger.SET_CHANNEL_NAME,
  onChange: Sync.parsedHandler(
    ranger.payloadZ,
    async ({ client, changed, params: { key }, onChange }) => {
      if (changed.key !== key) return;
      onChange(client.ranges.sugarOne(changed));
    },
  ),
};

export const use = Query.create({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => await client.ranges.retrieve(key),
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
  parent: parent ?? (await range.retrieveParent())?.key ?? "",
});

export interface UseFormQueryParams extends Optional<QueryParams, "key"> {}

export const useForm = (
  args: Pick<
    Query.UseFormArgs<UseFormQueryParams, typeof rangeFormSchema>,
    "initialValues" | "params" | "autoSave"
  >,
) =>
  Query.useForm<UseFormQueryParams, typeof rangeFormSchema>({
    ...args,
    name: "Range",
    schema: rangeFormSchema,
    retrieve: async ({ client, params: { key } }) => {
      if (key == null) return null;
      return await rangeToFormValues(await client.ranges.retrieve(key));
    },
    update: async ({ client, values, onChange }) => {
      const parentID = primitive.isZero(values.parent)
        ? undefined
        : ranger.ontologyID(values.parent as string);
      const rng = await client.ranges.create(values, { parent: parentID });
      await client.labels.label(rng.ontologyID, values.labels, { replace: true });
      onChange(await rangeToFormValues(rng, values.labels, values.parent));
    },
    listeners: [
      {
        channel: ranger.SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ranger.payloadZ,
          async ({ client, changed, onChange }) => {
            const values = await rangeToFormValues(client.ranges.sugarOne(changed));
            onChange((prev) => {
              if (prev?.key !== changed.key) return prev;
              return values;
            });
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ changed, onChange }) => {
            onChange((prev) => {
              if (prev == null) return prev;
              if (!Label.matchRelationship(changed, ranger.ontologyID(prev.key)))
                return prev;
              return {
                ...prev,
                labels: [
                  ...prev.labels.filter((l) => l !== changed.to.key),
                  changed.to.key,
                ],
              };
            });
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ changed, onChange }) => {
            onChange((prev) => {
              if (prev == null) return prev;
              const rel = ontology.relationShipZ.parse(changed);
              if (Label.matchRelationship(rel, ranger.ontologyID(prev.key)))
                return prev;
              return { ...prev, labels: prev.labels.filter((l) => l !== rel.to.key) };
            });
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ changed, onChange }) => {
            onChange((prev) => {
              if (prev == null) return prev;
              const rel = ontology.relationShipZ.parse(changed);
              if (
                rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE ||
                ontology.idsEqual(rel.to, ranger.ontologyID(prev.key))
              )
                return prev;
              return { ...prev, parent: rel.from.key };
            });
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ changed, onChange }) => {
            onChange((prev) => {
              if (prev == null) return prev;
              const rel = ontology.relationShipZ.parse(changed);
              if (
                rel.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE ||
                ontology.idsEqual(rel.to, ranger.ontologyID(prev.key))
              )
                return prev;
              return { ...prev, parent: undefined };
            });
          },
        ),
      },
    ],
  });

export const useLabels = (key: ranger.Key): UseReturn<label.Label[]> =>
  Label.useLabelsOf(ranger.ontologyID(key));
