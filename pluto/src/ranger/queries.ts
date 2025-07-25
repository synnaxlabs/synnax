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

export interface ChildrenParams {
  key: ranger.Key;
}

export const useChildren = Flux.createList<ChildrenParams, ranger.Key, ranger.Range>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => {
    const resources = await client.ontology.retrieveChildren(ranger.ontologyID(key));
    if (resources.length === 0) return [];
    return await client.ranges.retrieve({
      keys: resources.map(({ id: { key } }) => key),
      includeParent: true,
      includeLabels: true,
    });
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
        ontology.relationshipZ,
        async ({ changed, onChange, params, client }) => {
          if (!("key" in params)) return;
          if (!matchRelationshipAndID(changed, "to", ranger.ontologyID(params.key)))
            return;
          const range = await client.ranges.retrieve(changed.to.key);
          onChange(changed.to.key, range);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onDelete, params: { key } }) => {
          if (!matchRelationshipAndID(changed, "to", ranger.ontologyID(key))) return;
          onDelete(changed.to.key);
        },
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
        ontology.relationshipZ,
        async ({ changed, onChange, params: { key }, client }) =>
          matchRelationshipAndID(changed, "from", ranger.ontologyID(key)) &&
          onChange(await client.ranges.retrieve(key)),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
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

export interface QueryParams {
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
  ...ranger.payloadZ.omit({ timeRange: true }).partial({ key: true }).shape,
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
  labels: labels ?? (await range.retrieveLabels()).map((l) => l.key),
  parent: parent ?? (await range.retrieveParent())?.key ?? "",
});

export interface UseFormQueryParams extends Optional<QueryParams, "key"> {}

const ZERO_FORM_VALUES: z.infer<typeof rangeFormSchema> = {
  stage: "to_do",
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
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          onChange((prev) => {
            if (prev == null || prev.key == null) return prev;
            let next = prev;
            if (Label.matchRelationship(changed, ranger.ontologyID(prev.key)))
              next = {
                ...prev,
                labels: [
                  ...prev.labels.filter((l) => l !== changed.to.key),
                  changed.to.key,
                ],
              };
            if (
              changed.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
              ontology.idsEqual(changed.to, ranger.ontologyID(prev.key))
            )
              return { ...prev, parent: changed.from.key };
            return next;
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { key } }) => {
          if (key == null || !Label.matchRelationship(changed, ranger.ontologyID(key)))
            return;
          onChange((prev) => {
            if (prev == null) return prev;
            const nextLabels = prev.labels.filter((l) => l !== changed.to.key);
            return { ...prev, labels: nextLabels };
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

export interface ListParams
  extends Pick<
    ranger.RetrieveRequest,
    "includeLabels" | "includeParent" | "term" | "offset" | "limit"
  > {}

const DEFAULT_LIST_PARAMS: ranger.RetrieveRequest = {
  includeParent: true,
  includeLabels: true,
};

export const useList = Flux.createList<ListParams, ranger.Key, ranger.Range>({
  name: "Ranges",
  retrieve: async ({ client, params }) =>
    await client.ranges.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
    }),
  retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ranger.payloadZ,
        async ({ changed, onChange, client }) =>
          onChange(changed.key, client.ranges.sugarOne(changed)),
      ),
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

export interface ListKVParams {
  rangeKey: ranger.Key;
}

export const useListKV = Flux.createList<ListKVParams, string, ranger.KVPair>({
  name: "Range Meta Data",
  retrieve: async ({ client, params: { rangeKey } }) => {
    const kv = client.ranges.getKV(rangeKey);
    const pairs = await kv.list();
    return Object.entries(pairs).map(([key, value]) => ({
      key,
      value,
      range: rangeKey,
    }));
  },
  retrieveByKey: async ({ client, key, params: { rangeKey } }) => {
    if (rangeKey == null) return undefined;
    const kv = client.ranges.getKV(rangeKey);
    const value = await kv.get(key);
    return { key, value, range: rangeKey };
  },
  listeners: [
    {
      channel: ranger.KV_SET_CHANNEL,
      onChange: Sync.parsedHandler(ranger.kvPairZ, async ({ changed, onChange }) =>
        onChange(changed.key, changed),
      ),
    },
    {
      channel: ranger.KV_DELETE_CHANNEL,
      onChange: Sync.parsedHandler(ranger.kvPairZ, async ({ changed, onDelete }) =>
        onDelete(changed.key),
      ),
    },
  ],
});

export const useUpdateKV = Flux.createUpdate<ListKVParams, ranger.KVPair>({
  name: "Range Meta Data",
  update: async ({ client, value, onChange }) => {
    const kv = client.ranges.getKV(value.range);
    await kv.set(value.key, value.value);
    onChange(value);
  },
});

export interface UpdateParams {}

export const useUpdate = Flux.createUpdate<UpdateParams, ranger.Payload>({
  name: "Range",
  update: async ({ client, value, onChange }) =>
    onChange(await client.ranges.create(value)),
});

export const useDelete = Flux.createUpdate<UpdateParams, ranger.Key>({
  name: "Range",
  update: async ({ client, value }) => await client.ranges.delete(value),
});
