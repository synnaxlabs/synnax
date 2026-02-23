// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, ranger, type Synnax } from "@synnaxlabs/client";
import { array, type optional, primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { type List } from "@/list";
import { Ontology } from "@/ontology";
import { type ranger as aetherRanger } from "@/ranger/aether";
import { state } from "@/state";

export interface KVFluxStore extends Flux.UnaryStore<string, ranger.kv.Pair> {}
export interface AliasFluxStore extends Flux.UnaryStore<
  ranger.Key,
  ranger.alias.Alias
> {}

export const RANGE_KV_FLUX_STORE_KEY = "rangeKV";
export const RANGE_ALIASES_FLUX_STORE_KEY = "rangeAliases";

const RESOURCE_NAME = "range";
const PLURAL_RESOURCE_NAME = "ranges";
const KV_RESOURCE_NAME = "metadata";
const PLURAL_KV_RESOURCE_NAME = "metadata";
const PLURAL_CHILDREN_RESOURCE_NAME = "child ranges";
const PARENT_RESOURCE_NAME = "parent range";

export interface FluxSubStore extends Label.FluxSubStore, Ontology.FluxSubStore {
  [aetherRanger.FLUX_STORE_KEY]: aetherRanger.FluxStore;
  [RANGE_KV_FLUX_STORE_KEY]: KVFluxStore;
  [RANGE_ALIASES_FLUX_STORE_KEY]: AliasFluxStore;
}

export interface RetrieveQuery extends Pick<
  ranger.RetrieveRequest,
  "includeLabels" | "includeParent"
> {
  key: ranger.Key;
}

const BASE_QUERY: Partial<RetrieveQuery> = {
  includeParent: true,
  includeLabels: true,
};

const retrieveSingle = async ({
  client,
  store,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.ranges.get(key);
  if (cached != null) {
    const labels = Label.retrieveCachedLabelsOf(store, ranger.ontologyID(key));
    const parent = Ontology.retrieveCachedParentID(store, ranger.ontologyID(key));
    const next: ranger.Payload = { ...cached.payload, labels };
    if (parent != null) {
      const cached = store.ranges.get(parent.key);
      if (cached != null) next.parent = cached.payload;
    }
    return client.ranges.sugarOne(next);
  }
  const range = await client.ranges.retrieve({ ...BASE_QUERY, keys: [key] });
  const first = range[0];
  store.ranges.set(key, first);
  if (first.labels != null) {
    store.labels.set(first.labels);
    first.labels.forEach((l) => {
      const rel: ontology.Relationship = {
        from: ranger.ontologyID(key),
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(l.key),
      };
      store.relationships.set(ontology.relationshipToString(rel), rel);
    });
  }
  if (first.parent != null) {
    const rel: ontology.Relationship = {
      from: ranger.ontologyID(first.parent.key),
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: ranger.ontologyID(key),
    };
    store.relationships.set(ontology.relationshipToString(rel), rel);
  }

  return range[0];
};

const retrieveMultiple = async ({
  client,
  store,
  query: { keys },
}: Flux.RetrieveParams<RetrieveMultipleQuery, FluxSubStore>) => {
  const ranges: ranger.Range[] = [];
  const uncachedKeys: ranger.Key[] = [];

  // First, get all cached ranges
  for (const key of keys) {
    const cached = store.ranges.get(key);
    if (cached != null) {
      const labels = Label.retrieveCachedLabelsOf(store, ranger.ontologyID(key));
      const parent = Ontology.retrieveCachedParentID(store, ranger.ontologyID(key));
      const next: ranger.Payload = { ...cached.payload, labels };
      if (parent != null) {
        const cachedParent = store.ranges.get(parent.key);
        if (cachedParent != null) next.parent = cachedParent.payload;
      }
      ranges.push(client.ranges.sugarOne(next));
    } else uncachedKeys.push(key);
  }

  // Retrieve uncached ranges if any
  if (uncachedKeys.length > 0) {
    const uncachedRanges = await client.ranges.retrieve({
      ...BASE_QUERY,
      keys: uncachedKeys,
    });
    for (const range of uncachedRanges) {
      store.ranges.set(range.key, range);
      if (range.labels != null) store.labels.set(range.labels);
      ranges.push(range);
    }
  }

  return ranges;
};

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void => {
  const store = Flux.useStore();
  useEffect(() => store.ranges.onSet((c) => onSet(c.payload)), [store]);
};

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void => {
  const store = Flux.useStore();
  useEffect(() => store.ranges.onDelete((key) => onDelete(key)), [store]);
};

export interface ListChildrenQuery extends List.PagerParams {
  key?: ranger.Key;
}

const handleListLabelRelationshipSet = async (
  rel: ontology.Relationship,
  onChange: (key: ranger.Key, range: state.SetArg<ranger.Range | null>) => void,
  client: Synnax,
  store: FluxSubStore,
) => {
  const isLabel = ontology.matchRelationship(rel, {
    from: { type: "range" },
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
    to: { type: "label" },
  });
  if (isLabel) {
    let label = store.labels.get(rel.to.key);
    if (label == null) {
      label = await client.labels.retrieve({ key: rel.to.key });
      store.labels.set(rel.to.key, label);
    }
    onChange(rel.from.key, (prev) => {
      if (prev == null) return prev;
      return client.ranges.sugarOne({
        ...prev,
        labels: [...(prev.labels?.filter((l) => l.key !== rel.to.key) ?? []), label],
      });
    });
  }
};

const handleListParentRelationshipSet = async (
  rel: ontology.Relationship,
  onChange: (key: ranger.Key, range: state.SetArg<ranger.Range | null>) => void,
  client: Synnax,
  store: FluxSubStore,
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

export const useListChildren = Flux.createList<
  ListChildrenQuery,
  ranger.Key,
  ranger.Range,
  FluxSubStore
>({
  name: PLURAL_CHILDREN_RESOURCE_NAME,
  retrieve: async ({ client, query: { key }, store }) => {
    if (key == null) return [];
    const resources = await client.ontology.retrieveChildren(ranger.ontologyID(key), {
      types: ["range"],
    });
    if (resources.length === 0) return [];
    const query = { keys: resources.map(({ id: { key } }) => key) };
    return await retrieveMultiple({ client, store, query });
  },
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete, client, query: { key } }) => [
    store.ranges.onSet((range) => {
      onChange(range.key, (prev) => {
        if (prev == null) return prev;
        return client.ranges.sugarOne({
          ...range,
          parent: range.parent ?? prev.parent,
          labels: range.labels ?? prev.labels,
        });
      });
    }),
    store.ranges.onDelete(async (key) => onDelete(key)),
    store.relationships.onSet(async (rel) => {
      if (key == null) return;
      await handleListParentRelationshipSet(rel, onChange, client, store);
      await handleListLabelRelationshipSet(rel, onChange, client, store);
      const isChild = ontology.matchRelationship(rel, {
        from: ranger.ontologyID(key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "range" },
      });
      if (isChild) {
        const range = await client.ranges.retrieve({
          ...BASE_QUERY,
          keys: [rel.to.key],
        });
        return onChange(rel.to.key, range[0]);
      }
    }),
    store.relationships.onDelete(async (relKey) => {
      if (key == null) return;
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
            labels: prev.labels?.filter((l) => l.key !== rel.to.key),
          });
        });
    }),
  ],
});

export interface RetrieveParentQuery {
  id: ontology.ID;
}

export const {
  useRetrieve: useRetrieveParent,
  useRetrieveEffect: useRetrieveParentEffect,
} = Flux.createRetrieve<RetrieveParentQuery, ranger.Range | null, FluxSubStore>({
  name: PARENT_RESOURCE_NAME,
  retrieve: async ({ client, query: { id } }) => {
    const res = await client.ontology.retrieveParents(id);
    const parent = res.find(({ id: { type } }) => type === "range");
    if (parent == null) return null;
    return client.ranges.sugarOntologyResource(parent);
  },
  mountListeners: ({ store, onChange, client, query: { id } }) => [
    store.ranges.onSet((NextParent) => {
      onChange((prevParent) => {
        if (prevParent == null || prevParent.key !== NextParent.key) return prevParent;
        return client.ranges.sugarOne({
          ...NextParent,
          parent: NextParent.parent ?? prevParent.parent,
          labels: NextParent.labels ?? prevParent.labels,
        });
      });
    }),
    store.relationships.onSet(async (rel) => {
      const isParent = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: id,
      });
      if (!isParent) return;
      const parentIsRange = rel.from.type === "range";
      if (!parentIsRange) return onChange(null);
      const parent = await retrieveSingle({
        client,
        store,
        query: { key: rel.from.key },
      });
      onChange(client.ranges.sugarOne(parent.payload));
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      const isParent = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: id,
      });
      if (isParent) onChange(null);
      const isLabel = ontology.matchRelationship(rel, {
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: id,
      });
      if (isLabel)
        onChange((prev) => {
          if (prev == null) return prev;
          return client.ranges.sugarOne({
            ...prev,
            labels: prev.labels?.filter((l) => l.key !== rel.to.key),
          });
        });
    }),
  ],
});

export const { useRetrieve, useRetrieveObservable } = Flux.createRetrieve<
  RetrieveQuery,
  ranger.Range,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, onChange, client, query: { key } }) => [
    store.ranges.onSet(onChange, key),
    store.relationships.onSet(async (relationship) => {
      const isLabelChange = Label.matchRelationship(
        relationship,
        ranger.ontologyID(key),
      );
      if (isLabelChange) {
        const label = await Label.retrieveSingle({
          store,
          query: { key: relationship.to.key },
          client,
        });
        onChange(
          state.skipUndefined((prev) =>
            client.ranges.sugarOne({
              ...prev,
              labels: array.upsertKeyed(prev.labels, label),
            }),
          ),
        );
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
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      const otgID = ranger.ontologyID(key);
      const isLabelChange = Label.matchRelationship(rel, otgID);
      if (isLabelChange)
        return onChange(
          state.skipUndefined((p) =>
            client.ranges.sugarOne({
              ...p,
              labels: array.removeKeyed(p.labels, rel.to.key),
            }),
          ),
        );
      const isParentChange = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: otgID,
      });
      if (isParentChange)
        return onChange(
          state.skipUndefined((p) => client.ranges.sugarOne({ ...p, parent: null })),
        );
    }),
  ],
});

export interface RetrieveMultipleQuery {
  keys: ranger.Keys;
}

export const {
  useRetrieve: useRetrieveMultiple,
  useRetrieveObservable: useRetrieveObservableMultiple,
} = Flux.createRetrieve<RetrieveMultipleQuery, ranger.Range[], FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: retrieveMultiple,
  mountListeners: ({ store, onChange, client, query: { keys } }) => {
    const keysSet = new Set(keys);
    return [
      store.ranges.onSet(async (range) => {
        if (!keysSet.has(range.key)) return;
        onChange(
          state.skipUndefined((prev) =>
            prev.map((r) => (r.key === range.key ? range : r)),
          ),
        );
      }),
      store.ranges.onDelete(async (key) => {
        if (!keysSet.has(key)) return;
        onChange(state.skipUndefined((prev) => prev.filter((r) => r.key !== key)));
      }),
      store.relationships.onSet(async (relationship) => {
        for (const key of keys) {
          const isLabelChange = Label.matchRelationship(
            relationship,
            ranger.ontologyID(key),
          );
          if (isLabelChange) {
            const label = await client.labels.retrieve({ key: relationship.to.key });
            store.labels.set(relationship.to.key, label);
            onChange(
              state.skipUndefined((prev) =>
                prev.map((r) => {
                  if (r.key !== key) return r;
                  return client.ranges.sugarOne({
                    ...r,
                    labels: [
                      ...(r.labels ?? []).filter((l) => l.key !== label.key),
                      label,
                    ],
                  });
                }),
              ),
            );
          }
          const isParentChange = ontology.matchRelationship(relationship, {
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: ranger.ontologyID(key),
          });
          if (isParentChange) {
            const parent = await client.ranges.retrieve(relationship.from.key);
            store.ranges.set(relationship.from.key, parent);
            onChange(
              state.skipUndefined((prev) =>
                prev.map((r) => {
                  if (r.key !== key) return r;
                  return client.ranges.sugarOne({ ...r, parent });
                }),
              ),
            );
          }
        }
      }),
      store.relationships.onDelete(async (relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        for (const key of keys) {
          const isLabelChange = Label.matchRelationship(rel, ranger.ontologyID(key));
          if (isLabelChange)
            onChange(
              state.skipUndefined((prev) =>
                prev.map((r) => {
                  if (r.key !== key) return r;
                  return client.ranges.sugarOne({
                    ...r,
                    labels: array.removeKeyed(r.labels, rel.to.key),
                  });
                }),
              ),
            );

          const isParentChange = ontology.matchRelationship(rel, {
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: ranger.ontologyID(key),
          });
          if (isParentChange)
            onChange(
              state.skipUndefined((prev) =>
                prev.map((r) => {
                  if (r.key !== key) return r;
                  return client.ranges.sugarOne({ ...r, parent: null });
                }),
              ),
            );
        }
      }),
    ];
  },
});

export const formSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).partial({ key: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: z
    .object({ start: z.number(), end: z.number() })
    .refine(({ start, end }) => end >= start, {
      error: "End time must be after start time",
      path: ["end"],
    }),
});

export const toFormValues = (range: ranger.Range): z.infer<typeof formSchema> => ({
  ...range.payload,
  timeRange: range.timeRange.numeric,
  parent: range.parent?.key,
  labels: range.labels?.map((l) => l.key) ?? [],
});

export interface FormQuery extends optional.Optional<RetrieveQuery, "key"> {}

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key }, store, reset }) => {
    if (key == null) return;
    reset(toFormValues(await retrieveSingle({ client, store, query: { key } })));
  },
  update: async ({ client, value: getValue, reset, store, rollbacks }) => {
    const value = getValue();
    const parentKey = value.parent;
    const parentID = primitive.isNonZero(parentKey)
      ? ranger.ontologyID(parentKey)
      : undefined;
    const rng = await client.ranges.create(value, { parent: parentID });
    const labels = await Label.setLabelsFor({
      store,
      client,
      rollbacks,
      data: { id: rng.ontologyID, labels: value.labels },
    });
    let parent: ranger.Payload | null = null;
    if (primitive.isNonZero(parentKey))
      parent = (await retrieveSingle({ client, store, query: { key: parentKey } }))
        .payload;
    store.ranges.set(client.ranges.sugarOne({ ...rng.payload, labels, parent }));
    reset({
      ...value,
      ...rng.payload,
      timeRange: rng.timeRange.numeric,
      labels: value.labels,
      parent: value.parent,
    });
  },
  mountListeners: ({ store, reset, get, set }) => [
    store.ranges.onSet(async (range) => {
      const values = toFormValues(range);
      const prevKey = get<string>("key", { optional: true })?.value;
      if (prevKey == null || prevKey !== range.key) return;
      const prevParent = get<string>("parent", { optional: true })?.value;
      const prevLabels = get<string[]>("labels").value;
      reset({ ...values, labels: prevLabels, parent: prevParent });
    }),
    store.relationships.onSet((rel) => {
      const prevKey = get<string>("key", { optional: true })?.value;
      if (prevKey == null) return;
      const otgID = ranger.ontologyID(prevKey);
      const isLabelChange = Label.matchRelationship(rel, otgID);
      if (isLabelChange) {
        const prevLabels = get<string[]>("labels").value;
        return set("labels", [
          ...prevLabels.filter((l) => l !== rel.to.key),
          rel.to.key,
        ]);
      }
      const isParentChange = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: otgID,
      });
      if (isParentChange) set("parent", rel.from.key);
    }),
    store.relationships.onDelete((relKey) => {
      const prevKey = get<string>("key", { optional: true })?.value;
      if (prevKey == null) return;
      const rel = ontology.relationshipZ.parse(relKey);
      const otgID = ranger.ontologyID(prevKey);
      const isLabelChange = Label.matchRelationship(rel, otgID);
      if (isLabelChange)
        return set(
          "labels",
          get<string[]>("labels").value.filter((l) => l !== rel.to.key),
        );
      const isParentChange = ontology.matchRelationship(rel, {
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: otgID,
      });
      if (isParentChange) return set("parent", undefined);
    }),
  ],
});

export const useLabels = (
  key: ranger.Key,
): Flux.UseDirectRetrieveReturn<label.Label[]> =>
  Label.useRetrieveLabelsOf({ id: ranger.ontologyID(key) });

export interface ListQuery extends Omit<ranger.RetrieveRequest, "names"> {}

export const useList = Flux.createList<
  ListQuery,
  ranger.Key,
  ranger.Range,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store, query }) => {
    const keySet = primitive.isNonZero(query.keys) ? new Set(query.keys) : undefined;
    const hasLabelsSet = primitive.isNonZero(query.hasLabels)
      ? new Set(query.hasLabels)
      : undefined;
    return store.ranges.get((r) => {
      if (keySet != null && !keySet.has(r.key)) return false;
      if (
        hasLabelsSet != null &&
        (r.labels == null || !r.labels.some((l) => hasLabelsSet.has(l.key)))
      )
        return false;
      return true;
    });
  },
  retrieve: async ({ client, query }) =>
    await client.ranges.retrieve({ ...BASE_QUERY, ...query }),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({
    store,
    onChange,
    onDelete,
    client,
    query: { keys, hasLabels },
  }) => {
    const keysSet = primitive.isNonZero(keys) ? new Set(keys) : undefined;
    const hasLabelsSet =
      hasLabels != null && hasLabels.length > 0 ? new Set(hasLabels) : undefined;
    return [
      store.ranges.onSet((range) => {
        if (keysSet != null && !keysSet.has(range.key)) return;
        if (
          hasLabelsSet != null &&
          (range.labels == null || !range.labels.some((l) => hasLabelsSet.has(l.key)))
        )
          return;
        onChange(range.key, (prev) => {
          if (prev == null) return range;
          return client.ranges.sugarOne({
            ...range.payload,
            labels: range.labels ?? prev.labels,
          });
        });
      }),
      store.ranges.onDelete(onDelete),
      store.relationships.onSet(async (rel) => {
        await handleListParentRelationshipSet(rel, onChange, client, store);
        await handleListLabelRelationshipSet(rel, onChange, client, store);
      }),
      store.relationships.onDelete(async (relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        const isLabel = rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE;
        if (isLabel)
          onChange(rel.from.key, (prev) => {
            if (prev == null) return prev;
            return client.ranges.sugarOne({
              ...prev,
              labels: prev.labels?.filter((l) => l.key !== rel.to.key),
            });
          });
      }),
    ];
  },
});

export const metaDataFormSchema = z.object({
  pairs: z.array(z.object({ key: z.string(), value: z.string() })),
});

const deleteKVPairChannelValueZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

const SET_KV_LISTENER: Flux.ChannelListener<FluxSubStore, typeof ranger.kv.pairZ> = {
  channel: ranger.kv.SET_CHANNEL,
  schema: ranger.kv.pairZ,
  onChange: ({ store, changed }) => {
    store.rangeKV.set(ranger.kv.createPairKey(changed), changed);
  },
};

const DELETE_KV_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof deleteKVPairChannelValueZ
> = {
  channel: ranger.kv.DELETE_CHANNEL,
  schema: deleteKVPairChannelValueZ,
  onChange: ({ store, changed }) =>
    store.rangeKV.delete(ranger.kv.createPairKey(changed)),
};

export const KV_FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_KV_LISTENER, DELETE_KV_LISTENER],
};

export interface ListMetaDataQuery {
  rangeKey: ranger.Key;
}

export const useListMetaData = Flux.createList<
  ListMetaDataQuery,
  string,
  ranger.kv.Pair,
  FluxSubStore
>({
  name: PLURAL_KV_RESOURCE_NAME,
  retrieve: async ({ client, query: { rangeKey } }) => {
    const kv = client.ranges.getKV(rangeKey);
    const pairs = await kv.list();
    return Object.entries(pairs).map(([key, value]) => ({
      key,
      value,
      range: rangeKey,
    }));
  },
  retrieveByKey: async ({ client, key, query: { rangeKey } }) => {
    if (rangeKey == null) return undefined;
    const kv = client.ranges.getKV(rangeKey);
    const value = await kv.get(key);
    return { key, value, range: rangeKey };
  },
  mountListeners: ({ store, onChange, onDelete, query: { rangeKey } }) => [
    store.rangeKV.onSet((pair) => {
      if (pair.range !== rangeKey) return;
      onChange(pair.key, pair);
    }),
    store.rangeKV.onDelete((pairKey) => {
      const pair = deleteKVPairChannelValueZ.parse(pairKey);
      if (pair.range !== rangeKey) return;
      onDelete(pair.key);
    }),
  ],
});

export const kvPairFormSchema = ranger.kv.pairZ;

export interface KVFormQuery extends ListMetaDataQuery {}

const ZERO_KV_PAIR_FORM_VALUES: z.infer<typeof kvPairFormSchema> = {
  key: "",
  value: "",
  range: "",
};

export const useKVPairForm = Flux.createForm<
  KVFormQuery,
  typeof kvPairFormSchema,
  FluxSubStore
>({
  name: KV_RESOURCE_NAME,
  schema: kvPairFormSchema,
  retrieve: async () => undefined,
  initialValues: ZERO_KV_PAIR_FORM_VALUES,
  update: async ({ client, value: getPair, store }) => {
    const pair = getPair();
    const { key, value, range } = getPair();
    const kv = client.ranges.getKV(range);
    store.rangeKV.set(key, pair);
    await kv.set(key, value);
  },
});

export interface DeleteKVParams extends ListMetaDataQuery {
  key: string;
}

export const { useUpdate: useDeleteKV } = Flux.createUpdate<
  DeleteKVParams,
  FluxSubStore
>({
  name: KV_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store }) => {
    const { key, rangeKey } = data;
    const kv = client.ranges.getKV(rangeKey);
    await kv.delete(key);
    store.rangeKV.delete(key);
    return data;
  },
});

export interface SetKVParams extends ListMetaDataQuery, ranger.kv.Pair {}

export const { useUpdate: useUpdateKV } = Flux.createUpdate<SetKVParams, FluxSubStore>({
  name: KV_RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data }) => {
    const { range, key, value } = data;
    const kv = client.ranges.getKV(range);
    await kv.set(key, value);
    return data;
  },
});

export const { useUpdate: useCreate } = Flux.createUpdate<ranger.New, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store }) => {
    const rng = await client.ranges.create(data);
    store.ranges.set(rng.key, rng);
    return rng.payload;
  },
});

export type DeleteParams = ranger.Key | ranger.Keys;

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store }) => {
    await client.ranges.delete(data);
    store.ranges.delete(data);
    return data;
  },
});

const SET_ALIAS_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof ranger.alias.aliasZ
> = {
  channel: ranger.alias.SET_CHANNEL_NAME,
  schema: ranger.alias.aliasZ,
  onChange: ({ store, changed }) => {
    store.rangeAliases.set(ranger.alias.createKey(changed), changed);
  },
};
const aliasDeleteZ = z
  .string()
  .transform((val) => ranger.alias.decodeDeleteChange(val));

const DELETE_ALIAS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof aliasDeleteZ> = {
  channel: ranger.alias.DELETE_CHANNEL_NAME,
  schema: aliasDeleteZ,
  onChange: ({ store, changed }) =>
    store.rangeAliases.delete(ranger.alias.createKey(changed)),
};

export const ALIAS_FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_ALIAS_LISTENER, DELETE_ALIAS_LISTENER],
};

export interface RenameParams extends Pick<ranger.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, name } = data;
    rollbacks.push(
      store.ranges.set(
        key,
        state.skipUndefined((p) => client.ranges.sugarOne({ ...p, name })),
      ),
    );
    rollbacks.push(Ontology.renameFluxResource(store, ranger.ontologyID(key), name));
    await client.ranges.rename(key, name);
    return data;
  },
});
