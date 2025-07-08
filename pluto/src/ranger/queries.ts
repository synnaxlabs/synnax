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

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";
import { Label } from "@/label";
import { matchRelationshipAndID } from "@/ontology/queries";

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

export interface ChildrenParams extends Flux.Params {
  key: ranger.Key;
}

export const useChildren = Flux.createList<ChildrenParams, ranger.Key, ranger.Range>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => {
    const resources = await client.ontology.retrieveChildren(ranger.ontologyID(key));
    return resources
      .filter(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE)
      .map((resource) => client.ranges.sugarOntologyResource(resource));
  },
  retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ranger.payloadZ,
        async ({ changed, onChange, client }) => {
          onChange(changed.key, (prev) => {
            if (prev == null) return prev;
            return client.ranges.sugarOne(changed);
          });
        },
      ),
    },
    {
      channel: ranger.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(ranger.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationShipZ,
        async ({ changed, onChange, params, client }) => {
          if (!("key" in params)) return;
          if (!matchRelationshipAndID(changed, "to", ranger.ontologyID(params.key)))
            return;
          const range = await client.ranges.retrieve(params.key);
          onChange(params.key, range);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationShipZ,
        async ({ changed, onChange, params: { key }, client }) =>
          matchRelationshipAndID(changed, "to", ranger.ontologyID(key)) &&
          onChange(key, await client.ranges.retrieve(key)),
      ),
    },
  ],
});

export const retrieveParent = Flux.createRetrieve<
  { key: ranger.Key },
  ranger.Range | null
>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => {
    const res = await client.ontology.retrieveParents(ranger.ontologyID(key));
    const parent = res.find(({ id: { type } }) => type === ranger.ONTOLOGY_TYPE);
    if (parent == null) return null;
    return client.ranges.sugarOntologyResource(parent);
  },
  listeners: [
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationShipZ,
        async ({ changed, onChange, params: { key }, client }) =>
          matchRelationshipAndID(changed, "from", ranger.ontologyID(key)) &&
          onChange(await client.ranges.retrieve(key)),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationShipZ,
        async ({ changed, onChange, params: { key }, client }) =>
          matchRelationshipAndID(changed, "from", ranger.ontologyID(key)) &&
          onChange(await client.ranges.retrieve(key)),
      ),
    },
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ranger.payloadZ,
        async ({ changed, onChange, params: { key }, client }) =>
          changed.key === key && onChange(client.ranges.sugarOne(changed)),
      ),
    },
  ],
});

export interface QueryParams extends Flux.Params {
  key: ranger.Key;
}

const SET_LISTENER_CONFIG: Flux.RetrieveListenerConfig<QueryParams, ranger.Range> = {
  channel: ranger.SET_CHANNEL_NAME,
  onChange: Sync.parsedHandler(
    ranger.payloadZ,
    async ({ client, changed, params: { key }, onChange }) => {
      if (changed.key !== key) return;
      onChange(client.ranges.sugarOne(changed));
    },
  ),
};

export const retrieveQuery = Flux.createRetrieve<QueryParams, ranger.Range>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => await client.ranges.retrieve(key),
  listeners: [SET_LISTENER_CONFIG],
});

export const useRetrieve = retrieveQuery.useDirect;

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

const ZERO_FORM_VALUES: z.infer<typeof rangeFormSchema> = {
  key: "",
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

export const useForm = Flux.createForm<UseFormQueryParams, typeof rangeFormSchema>({
  name: "Range",
  schema: rangeFormSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return null;
    return await rangeToFormValues(await client.ranges.retrieve(key));
  },
  update: async ({ client, value, onChange }) => {
    const parentID = primitive.isZero(value.parent)
      ? undefined
      : ranger.ontologyID(value.parent as string);
    const rng = await client.ranges.create(value, { parent: parentID });
    await client.labels.label(rng.ontologyID, value.labels, { replace: true });
    onChange(await rangeToFormValues(rng, value.labels, value.parent));
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
            if (Label.matchRelationship(rel, ranger.ontologyID(prev.key))) return prev;
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

export const useLabels = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<label.Label[]> =>
  Label.retrieveLabelsOf.useDirect({ params: { id: ranger.ontologyID(key) } });

export interface ListParams extends Flux.Params {
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
      onChange: Sync.parsedHandler(ranger.payloadZ, async ({ changed, onChange }) => {
        onChange(changed.key, (prev) => {
          if (prev == null) return prev;
          return changed;
        });
      }),
    },
    {
      channel: ranger.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(ranger.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
  ],
});

export const metaDataFormSchema = z.object({
  pairs: z.array(z.object({ key: z.string(), value: z.string() })),
});

export interface MetaDataFormParams extends Flux.Params {
  key: ranger.Key;
}

export const useMetaDataForm = Flux.createList<
  MetaDataFormParams,
  string,
  ranger.KVPair
>({
  retrieve: async ({ client, params: { key } }) => {
    const kv = client.ranges.getKV(key);
    const pairs = await kv.list();
    return Object.entries(pairs).map(([key, value]) => ({ key, value }));
  },
  retrieveByKey: async ({ client, key }) => ({
    key: await client.ranges.getKV(key).get(key),
    value: "",
    range: key,
  }),
  listeners: [
});
