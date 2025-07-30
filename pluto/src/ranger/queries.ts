// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, ranger } from "@synnaxlabs/client";
import {
  type MultiSeries,
  type Optional,
  primitive,
  type TelemValue,
} from "@synnaxlabs/x";
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

const handleLabelRelationshipSet: Sync.ListenerHandler<
  ontology.Relationship,
  Flux.ListListenerExtraArgs<{}, string, ranger.Range>
> = async ({ changed, onChange, client }) => {
  const isLabel = ontology.matchRelationship(changed, {
    from: { type: ranger.ONTOLOGY_TYPE },
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
    to: { type: label.ONTOLOGY_TYPE },
  });
  if (isLabel) {
    const label = await client.labels.retrieve({ key: changed.to.key });
    onChange(changed.from.key, (prev) => {
      if (prev == null) return prev;
      return client.ranges.sugarOne({
        ...prev,
        labels: [...prev.labels, label],
      });
    });
  }
};

const handleParentRelationshipSet: Sync.ListenerHandler<
  ontology.Relationship,
  Flux.ListListenerExtraArgs<{}, string, ranger.Range>
> = async ({ changed, onChange, client }) => {
  const isParent = ontology.matchRelationship(changed, {
    from: { type: ranger.ONTOLOGY_TYPE },
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: ranger.ONTOLOGY_TYPE },
  });
  if (isParent) {
    const parent = await client.ranges.retrieve(changed.from.key);
    onChange(changed.to.key, (prev) => {
      if (prev == null) return prev;
      return client.ranges.sugarOne({ ...prev, parent });
    });
  }
};

export const useChildren = Flux.createList<ChildrenParams, ranger.Key, ranger.Range>({
  name: "Range",
  retrieve: async ({ client, params: { key } }) => {
    const resources = await client.ontology.retrieveChildren(ranger.ontologyID(key), {
      types: [ranger.ONTOLOGY_TYPE],
    });
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
            // If the range doesn't exist in the list, don't add it, as it may not
            // be a child of the range.
            if (prev == null) return prev;
            return client.ranges.sugarOne({
              ...prev.payload,
              ...changed,
              parent: prev.parent ?? changed.parent,
              labels: prev.labels ?? [],
            });
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
      onChange: Sync.parsedHandler(ontology.relationshipZ, async (args) => {
        const { changed, onChange, client, params } = args;
        if (!("key" in params)) return;
        const isChild = ontology.matchRelationship(changed, {
          from: ranger.ontologyID(params.key),
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: { type: ranger.ONTOLOGY_TYPE },
        });
        if (isChild) {
          const range = await client.ranges.retrieve({
            keys: [changed.to.key],
            includeParent: true,
            includeLabels: true,
          });
          return onChange(changed.to.key, range[0]);
        }
        await handleLabelRelationshipSet(args);
        await handleParentRelationshipSet(args);
      }),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onDelete, onChange, client, params: { key } }) => {
          const isChild = matchRelationshipAndID(changed, "to", ranger.ontologyID(key));
          if (isChild) return onDelete(changed.to.key);
          const isLabel = changed.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE;
          if (isLabel)
            onChange(changed.from.key, (prev) => {
              if (prev == null) return prev;
              return client.ranges.sugarOne({
                ...prev,
                labels: prev.labels.filter((l) => l.key !== changed.to.key),
              });
            });
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
          onChange(await client.ranges.retrieve(changed.from.key)),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { key }, client }) =>
          matchRelationshipAndID(changed, "from", ranger.ontologyID(key)) &&
          onChange(await client.ranges.retrieve(changed.from.key)),
      ),
    },
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ranger.payloadZ,
        async ({ changed, onChange, client }) => {
          onChange((prev) => {
            if (prev == null || prev.key !== changed.key) return prev;
            return client.ranges.sugarOne({ ...prev.payload, ...changed });
          });
        },
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
  stage: "to_do",
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
    onChange({
      ...value,
      ...rng.payload,
      timeRange: rng.timeRange.numeric,
      labels: value.labels,
      parent: value.parent,
    });
  },
  listeners: [
    {
      channel: ranger.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ranger.payloadZ,
        async ({ client, changed, onChange }) => {
          const values = await toFormValues(client.ranges.sugarOne(changed));
          onChange((prev) => {
            if (prev?.key !== changed.key) return prev;
            return { ...values, labels: prev.labels, parent: prev.parent };
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
            const otgID = ranger.ontologyID(prev.key);
            const isLabelChange = Label.matchRelationship(changed, otgID);
            if (isLabelChange)
              return {
                ...prev,
                labels: [
                  ...prev.labels.filter((l) => l !== changed.to.key),
                  changed.to.key,
                ],
              };

            const isParentChange = ontology.matchRelationship(changed, {
              type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
              to: otgID,
            });
            if (isParentChange) return { ...prev, parent: changed.from.key };
            return prev;
          });
        },
      ),
    },
    // {
    //   channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
    //   onChange: Sync.parsedHandler(
    //     ontology.relationshipZ,
    //     async ({ changed, onChange, params: { key } }) => {
    //       if (key == null || !Label.matchRelationship(changed, ranger.ontologyID(key)))
    //         return;
    //       onChange((prev) => {
    //         if (prev == null) return prev;
    //         const nextLabels = prev.labels.filter((l) => l !== changed.to.key);
    //         return { ...prev, labels: nextLabels };
    //       });
    //     },
    //   ),
    // },
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
          onChange(changed.key, (prev) => {
            const next = {
              ...prev?.payload,
              ...changed,
              parent: prev?.parent ?? changed.parent,
              labels: prev?.labels ?? [],
            };
            return client.ranges.sugarOne(next);
          }),
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
      onChange: Sync.parsedHandler(ontology.relationshipZ, async (args) => {
        await handleLabelRelationshipSet(args);
        await handleParentRelationshipSet(args);
      }),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, client }) => {
          if (changed.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE)
            return onChange(changed.from.key, (prev) => {
              if (prev == null) return prev;
              return client.ranges.sugarOne({
                ...prev,
                labels: prev.labels.filter((l) => l.key !== changed.to.key),
              });
            });
        },
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

export const useDelete = Flux.createUpdate<UpdateParams, ranger.Key | ranger.Keys>({
  name: "Range",
  update: async ({ client, value }) => await client.ranges.delete(value),
});
