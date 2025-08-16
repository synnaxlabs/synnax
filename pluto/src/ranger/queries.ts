// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, ranger, type Synnax } from "@synnaxlabs/client";
import { array, type Optional, primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { type Ontology } from "@/ontology";
import { type ranger as aetherRanger } from "@/ranger/aether";
import { type state } from "@/state";

export interface KVFluxStore extends Flux.UnaryStore<string, ranger.KVPair> {}

export interface AliasFluxStore extends Flux.UnaryStore<ranger.Key, ranger.Alias> {}

interface SubStore extends Flux.Store {
  ranges: aetherRanger.FluxStore;
  relationships: Ontology.RelationshipFluxStore;
  labels: Label.FluxStore;
  rangeKV: Flux.UnaryStore<string, ranger.KVPair>;
  rangeAliases: AliasFluxStore;
}

const cachedRetrieve = async (client: Synnax, store: SubStore, key: ranger.Key) => {
  const cached = store.ranges.get(key);
  if (cached != null) return cached;
  const range = await client.ranges.retrieve(key);
  store.ranges.set(key, range);
  return range;
};

const multiCachedRetrieve = async (
  client: Synnax,
  store: SubStore,
  params: ranger.RetrieveRequest,
) => {
  const ranges = await client.ranges.retrieve({
    ...params,
    includeParent: true,
    includeLabels: true,
  });
  ranges.forEach((range) => store.ranges.set(range.key, range));
  return ranges;
};

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => {
    const destructor = store.ranges.onSet(async (changed) => onSet(changed));
    return () => destructor();
  }, [store.ranges]);
};

export interface ChildrenParams {
  key: ranger.Key;
}

const handleListLabelRelationshipSet = async (
  rel: ontology.Relationship,
  onChange: (key: ranger.Key, range: state.SetArg<ranger.Range | null>) => void,
  client: Synnax,
  store: SubStore,
) => {
  const isLabel = ontology.matchRelationship(rel, {
    from: { type: "range" },
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
    to: { type: "label" },
  });
  if (isLabel) {
    const label = await client.labels.retrieve({ key: rel.to.key });
    store.labels.set(rel.to.key, label);
    onChange(rel.from.key, (prev) => {
      if (prev == null) return prev;
      return client.ranges.sugarOne({
        ...prev,
        labels: [...prev.labels.filter((l) => l.key !== rel.to.key), label],
      });
    });
  }
};

const handleListParentRelationshipSet = async (
  rel: ontology.Relationship,
  onChange: (key: ranger.Key, range: state.SetArg<ranger.Range | null>) => void,
  client: Synnax,
  store: SubStore,
) => {
  const isParent = ontology.matchRelationship(rel, {
    from: { type: "range" },
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "range" },
  });
  if (isParent) {
    const parent = await client.ranges.retrieve(rel.from.key);
    store.ranges.set(rel.from.key, parent);
    onChange(rel.to.key, (prev) => {
      if (prev == null) return prev;
      return client.ranges.sugarOne({ ...prev, parent });
    });
  }
};

export const useChildren = Flux.createList<
  ChildrenParams,
  ranger.Key,
  ranger.Range,
  SubStore
>({
  name: "Range",
  retrieve: async ({ client, params: { key }, store }) => {
    const resources = await client.ontology.retrieveChildren(ranger.ontologyID(key), {
      types: ["range"],
    });
    return await multiCachedRetrieve(client, store, {
      keys: resources.map(({ id: { key } }) => key),
    });
  },
  retrieveByKey: async ({ client, key, store }) =>
    await cachedRetrieve(client, store, key),
  mountListeners: ({ store, onChange, onDelete, client, params: { key } }) => [
    store.ranges.onSet(async (range) => {
      onChange(range.key, (prev) => {
        if (prev == null) return prev;
        return client.ranges.sugarOne({
          ...prev,
          parent: range.parent ?? prev.parent,
          labels: range.labels ?? prev.labels,
        });
      });
    }),
    store.ranges.onDelete(async (key) => onDelete(key)),
    store.relationships.onSet(async (rel) => {
      await handleListParentRelationshipSet(rel, onChange, client, store);
      await handleListLabelRelationshipSet(rel, onChange, client, store);
      const isChild = ontology.matchRelationship(rel, {
        from: ranger.ontologyID(key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "range" },
      });
      if (isChild) {
        const range = await client.ranges.retrieve({
          keys: [rel.to.key],
          includeParent: true,
          includeLabels: true,
        });
        return onChange(rel.to.key, range[0]);
      }
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      const isChild = ontology.matchRelationship(rel, {
        from: ranger.ontologyID(key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "range" },
      });
      if (isChild) return onDelete(rel.to.key);
      const isLabel = rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE;
      if (isLabel)
        onChange(rel.from.key, (prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({
            ...prev,
            labels: prev.labels.filter((l) => l.key !== rel.to.key),
          });
        });
    }),
  ],
});

export const retrieveParent = Flux.createRetrieve<
  { key: ranger.Key },
  ranger.Range | null,
  SubStore
>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => {
    const res = await client.ontology.retrieveParents(ranger.ontologyID(key));
    const parent = res.find(({ id: { type } }) => type === "range");
    if (parent == null) return null;
    return client.ranges.sugarOntologyResource(parent);
  },
  mountListeners: ({ store, onChange, client, params: { key } }) => [
    store.ranges.onSet(async (range) => {
      onChange((prev) => {
        if (prev == null || prev.key !== key) return prev;
        return client.ranges.sugarOne({ ...prev, parent: range.parent ?? prev.parent });
      });
    }),
    store.relationships.onSet(async (rel) => {
      const isParent = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ranger.ontologyID(key),
      });
      if (isParent) {
        const parent = await cachedRetrieve(client, store, rel.from.key);
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({ ...prev, parent });
        });
      }
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      const isParent = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ranger.ontologyID(key),
      });
      if (isParent)
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({ ...prev, parent: null });
        });
      const isLabel = ontology.matchRelationship(rel, {
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: ranger.ontologyID(key),
      });
      if (isLabel)
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({
            ...prev,
            labels: prev.labels.filter((l) => l.key !== rel.to.key),
          });
        });
    }),
  ],
});

export interface QueryParams {
  key: ranger.Key;
}

export const retrieveQuery = Flux.createRetrieve<QueryParams, ranger.Range, SubStore>({
  name: "Range",
  retrieve: async ({ client, params: { key }, store }) =>
    await cachedRetrieve(client, store, key),
  mountListeners: ({ store, onChange, client, params: { key } }) => [
    store.ranges.onSet(async (range) => {
      if (range != null) return onChange(range);
    }, key),
    store.relationships.onSet(async (relationship) => {
      const isLabelChange = Label.matchRelationship(
        relationship,
        ranger.ontologyID(key),
      );
      if (isLabelChange) {
        const label = await client.labels.retrieve({ key: relationship.to.key });
        store.labels.set(relationship.to.key, label);
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({
            ...prev,
            labels: [...(prev.labels ?? []).filter((l) => l.key !== label.key), label],
          });
        });
      }
      const isParentChange = ontology.matchRelationship(relationship, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ranger.ontologyID(key),
      });
      if (isParentChange) {
        const parent = await client.ranges.retrieve(relationship.from.key);
        store.ranges.set(relationship.from.key, parent);
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({ ...prev, parent });
        });
      }
    }, key),
  ],
});

export const useRetrieve = retrieveQuery.useDirect;

export const formSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).partial({ key: true }).shape,
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
  labels: labels ?? (await range.retrieveLabels()).map((l) => l.key),
  parent: parent ?? (await range.retrieveParent())?.key ?? "",
});

export interface UseFormQueryParams extends Optional<QueryParams, "key"> {}

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

export const useForm = Flux.createForm<UseFormQueryParams, typeof formSchema, SubStore>(
  {
    name: "Range",
    schema: formSchema,
    initialValues: ZERO_FORM_VALUES,
    retrieve: async ({ client, params: { key }, store }) => {
      if (key == null) return null;
      return await toFormValues(await cachedRetrieve(client, store, key));
    },
    update: async ({ client, value, onChange, store }) => {
      const parentID = primitive.isZero(value.parent)
        ? undefined
        : ranger.ontologyID(value.parent as string);
      const rng = await client.ranges.create(value, { parent: parentID });
      await client.labels.label(rng.ontologyID, value.labels, { replace: true });
      store.ranges.set(rng.key, rng);
      onChange({
        ...value,
        ...rng.payload,
        timeRange: rng.timeRange.numeric,
        labels: value.labels,
        parent: value.parent,
      });
    },
    mountListeners: ({ store, onChange, params: { key } }) => [
      store.ranges.onSet(async (range) => {
        const values = await toFormValues(range);
        onChange((prev) => {
          if (prev == null || prev?.key !== key) return prev;
          return { ...values, labels: prev.labels, parent: prev.parent };
        });
      }),
      store.relationships.onSet(async (rel) => {
        onChange((prev) => {
          if (prev == null || prev.key == null) return prev;
          const otgID = ranger.ontologyID(prev.key);
          const isLabelChange = Label.matchRelationship(rel, otgID);
          if (isLabelChange)
            return {
              ...prev,
              labels: [...prev.labels.filter((l) => l !== rel.to.key), rel.to.key],
            };
          const isParentChange = ontology.matchRelationship(rel, {
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: otgID,
          });
          if (isParentChange) return { ...prev, parent: rel.from.key };
          return prev;
        });
      }),
      store.relationships.onDelete(async (relKey) => {
        onChange((prev) => {
          if (prev == null || prev.key == null) return prev;
          const rel = ontology.relationshipZ.parse(relKey);
          const otgID = ranger.ontologyID(prev.key);
          const isLabelChange = Label.matchRelationship(rel, otgID);
          if (isLabelChange)
            return {
              ...prev,
              labels: prev.labels.filter((l) => l !== rel.to.key),
            };
          const isParentChange = ontology.matchRelationship(rel, {
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: otgID,
          });
          if (isParentChange) return { ...prev, parent: undefined };
          return prev;
        });
      }),
    ],
  },
);

export const useLabels = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<label.Label[]> =>
  Label.retrieveLabelsOf.useDirect({ params: { id: ranger.ontologyID(key) } });

export interface ListParams
  extends Pick<
    ranger.RetrieveRequest,
    "includeLabels" | "includeParent" | "searchTerm" | "offset" | "limit"
  > {}

const DEFAULT_LIST_PARAMS: ranger.RetrieveRequest = {
  includeParent: true,
  includeLabels: true,
};

export const useList = Flux.createList<ListParams, ranger.Key, ranger.Range, SubStore>({
  name: "Ranges",
  retrieve: async ({ client, params }) =>
    await client.ranges.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
    }),
  retrieveByKey: async ({ client, key, store }) => {
    const cached = store.ranges.get(key);
    if (cached != null) return cached;
    const range = await client.ranges.retrieve(key);
    store.ranges.set(key, range);
    return range;
  },
  mountListeners: ({ store, onChange, onDelete, client }) => [
    store.ranges.onSet(async (range) => {
      onChange(range.key, (prev) => {
        if (prev == null) return prev;
        return client.ranges.sugarOne({
          ...prev,
          parent: range.parent ?? prev.parent,
          labels: range.labels ?? prev.labels,
        });
      });
    }),
    store.ranges.onDelete(async (key) => onDelete(key)),
    store.relationships.onSet(async (rel) => {
      await handleListParentRelationshipSet(rel, onChange, client, store);
      await handleListLabelRelationshipSet(rel, onChange, client, store);
    }),
  ],
});

export const metaDataFormSchema = z.object({
  pairs: z.array(z.object({ key: z.string(), value: z.string() })),
});

export interface ListKVParams {
  rangeKey: ranger.Key;
}

const deleteKVPairChannelValueZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

const SET_KV_LISTENER: Flux.ChannelListener<SubStore, typeof ranger.kvPairZ> = {
  channel: ranger.KV_SET_CHANNEL,
  schema: ranger.kvPairZ,
  onChange: async ({ store, changed }) => {
    store.rangeKV.set(`${changed.range}<--->${changed.key}`, changed);
  },
};

const DELETE_KV_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof deleteKVPairChannelValueZ
> = {
  channel: ranger.KV_DELETE_CHANNEL,
  schema: deleteKVPairChannelValueZ,
  onChange: async ({ store, changed }) => {
    store.rangeKV.delete(`${changed.range}<--->${changed.key}`);
  },
};

export const KV_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_KV_LISTENER, DELETE_KV_LISTENER],
};

export const useListKV = Flux.createList<ListKVParams, string, ranger.KVPair, SubStore>(
  {
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
    mountListeners: ({ store, onChange, onDelete, params: { rangeKey } }) => [
      store.rangeKV.onSet(async (pair) => {
        if (pair.range !== rangeKey) return;
        onChange(pair.key, pair);
      }),
      store.rangeKV.onDelete(async (pairKey) => {
        const pair = deleteKVPairChannelValueZ.parse(pairKey);
        if (pair.range !== rangeKey) return;
        onDelete(pair.key);
      }),
    ],
  },
);

export const kvPairFormSchema = ranger.kvPairZ;

export const useKVPairForm = Flux.createForm<
  ListKVParams,
  typeof kvPairFormSchema,
  SubStore
>({
  name: "Range Meta Data",
  schema: kvPairFormSchema,
  retrieve: async () => null,
  update: async ({ client, value }) => {
    const kv = client.ranges.getKV(value.range);
    await kv.set(value.key, value.value);
  },
  initialValues: {
    key: "",
    value: "",
    range: "",
  },
});

export const useDeleteKV = Flux.createUpdate<ListKVParams, string>({
  name: "Range Meta Data",
  update: async ({ client, value, params: { rangeKey } }) => {
    const kv = client.ranges.getKV(rangeKey);
    await kv.delete(value);
  },
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

export const useUpdate = Flux.createUpdate<UpdateParams, ranger.Payload, SubStore>({
  name: "Range",
  update: async ({ client, value, onChange, store }) => {
    const rng = await client.ranges.create(value);
    store.ranges.set(rng.key, rng);
    onChange(rng);
  },
});

export const useDelete = Flux.createUpdate<
  UpdateParams,
  ranger.Key | ranger.Keys,
  SubStore
>({
  name: "Range",
  update: async ({ client, value, store }) => {
    await client.ranges.delete(value);
    array.toArray(value).forEach((key) => store.ranges.delete(key));
  },
});

const SET_ALIAS_LISTENER: Flux.ChannelListener<SubStore, typeof ranger.aliasZ> = {
  channel: ranger.SET_ALIAS_CHANNEL_NAME,
  schema: ranger.aliasZ,
  onChange: async ({ store, changed }) => {
    store.rangeAliases.set(ranger.aliasKey(changed), changed);
  },
};

const DELETE_ALIAS_LISTENER: Flux.ChannelListener<SubStore, typeof ranger.aliasZ> = {
  channel: ranger.DELETE_ALIAS_CHANNEL_NAME,
  schema: ranger.aliasZ,
  onChange: async ({ store, changed }) =>
    store.rangeAliases.delete(ranger.aliasKey(changed)),
};

export const ALIAS_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_ALIAS_LISTENER, DELETE_ALIAS_LISTENER],
};
