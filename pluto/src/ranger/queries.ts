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
import { z } from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { Ontology } from "@/ontology";
import { Synnax } from "@/synnax";

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Flux.useListener({
    channel: ranger.SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ranger.payloadZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void =>
  Flux.useListener({
    channel: ranger.DELETE_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ranger.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });

export const useAliasSetSynchronizer = (onSet: (alias: ranger.Alias) => void): void =>
  Flux.useListener({
    channel: ranger.SET_ALIAS_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ranger.aliasZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void =>
  Flux.useListener({
    channel: ranger.DELETE_ALIAS_CHANNEL_NAME,
    onChange: Flux.stringHandler(async (args) => {
      onDelete(ranger.decodeDeleteAliasChange(args.changed));
    }),
  });

export const useChildren = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<ranger.Range[]> => {
  const res = Ontology.useChildren(ranger.ontologyID(key));
  const client = Synnax.use();
  if (res.variant !== "success") return { ...res, data: null };
  if (client == null) return { ...res, data: [] };
  return {
    ...res,
    data: res.data
      .filter(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE)
      .map((child) => client.ranges.sugarOntologyResource(child)),
  };
};

export const useParent = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<ranger.Range | null> => {
  const res = Ontology.useParents(ranger.ontologyID(key));
  const client = Synnax.use();
  if (res.variant !== "success") return { ...res, data: null };
  const parent = res.data.find(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE);
  if (parent == null || client == null) return { ...res, data: null };
  return { ...res, data: client.ranges.sugarOntologyResource(parent) };
};

export interface QueryParams {
  key: ranger.Key;
}

const SET_LISTENER_CONFIG: Flux.RetrieveListenerConfig<QueryParams, ranger.Range> = {
  channel: ranger.SET_CHANNEL_NAME,
  onChange: Flux.parsedHandler(
    ranger.payloadZ,
    async ({ client, changed, params: { key }, onChange }) => {
      if (changed.key !== key) return;
      onChange(client.ranges.sugarOne(changed));
    },
  ),
};

export const retrieve = Flux.createRetrieve<QueryParams, ranger.Range>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => await client.ranges.retrieve(key),
  listeners: [SET_LISTENER_CONFIG],
});

export const formSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: z.object({ start: z.number(), end: z.number() }),
});

export const toFormValues = async (
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

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  key: "",
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

export const useForm = Flux.createForm<UseFormQueryParams, typeof formSchema>({
  name: "Range",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return null;
    return await toFormValues(await client.ranges.retrieve(key));
  },
  update: async ({ client, value, onChange }) => {
    const parentID = primitive.isZero(value.parent)
      ? undefined
      : ranger.ontologyID(value.parent as string);
    const rng = await client.ranges.create(value, { parent: parentID });
    await client.labels.label(rng.ontologyID, value.labels, { replace: true });
    onChange(await toFormValues(rng, value.labels, value.parent));
  },
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ranger.payloadZ,
        async ({ client, changed, onChange }) => {
          const values = await toFormValues(client.ranges.sugarOne(changed));
          onChange((prev) => {
            if (prev?.key !== changed.key) return prev;
            return values;
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
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
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          onChange((prev) => {
            if (prev == null) return prev;
            if (!Label.matchRelationship(changed, ranger.ontologyID(prev.key)))
              return prev;
            return {
              ...prev,
              labels: prev.labels.filter((l) => l !== changed.to.key),
            };
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          onChange((prev) => {
            if (prev == null) return prev;
            if (
              changed.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE ||
              ontology.idsEqual(changed.to, ranger.ontologyID(prev.key))
            )
              return prev;
            return { ...prev, parent: changed.from.key };
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          onChange((prev) => {
            if (prev == null) return prev;
            if (
              changed.type !== ontology.PARENT_OF_RELATIONSHIP_TYPE ||
              ontology.idsEqual(changed.to, ranger.ontologyID(prev.key))
            )
              return prev;
            return { ...prev, parent: undefined };
          });
        },
      ),
    },
  ],
});

export const useLabels = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<label.Label[]> =>
  Label.retrieveLabelsOf.useDirect({ params: { id: ranger.ontologyID(key) } });

interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, ranger.Key, ranger.Payload>({
  name: "Range",
  retrieve: async ({ client, params }) => await client.ranges.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(ranger.payloadZ, async ({ changed, onChange }) => {
        onChange(changed.key, (prev) => {
          if (prev == null) return prev;
          return changed;
        });
      }),
    },
    {
      channel: ranger.DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(ranger.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
  ],
});
