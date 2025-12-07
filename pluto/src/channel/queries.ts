// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, type group, ontology, ranger } from "@synnaxlabs/client";
import { array, deep, type optional, primitive, TimeSpan } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { type Group } from "@/group";
import { Ontology } from "@/ontology";
import { type Ranger } from "@/ranger";
import { state } from "@/state";

export const FLUX_STORE_KEY = "channels";
const RESOURCE_NAME = "channel";
const PLURAL_RESOURCE_NAME = "channels";

export interface FluxStore extends Flux.UnaryStore<channel.Key, channel.Channel> {}

interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.AliasFluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [Group.FLUX_STORE_KEY]: Group.FluxStore;
}

const SET_CHANNEL_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof channel.payloadZ
> = {
  channel: channel.SET_CHANNEL_NAME,
  schema: channel.payloadZ,
  onChange: async ({ store, changed, client }) =>
    store.channels.set(client.channels.sugar(changed)),
};

const DELETE_CHANNEL_LISTENER: Flux.ChannelListener<FluxSubStore, typeof channel.keyZ> =
  {
    channel: channel.DELETE_CHANNEL_NAME,
    schema: channel.keyZ,
    onChange: ({ store, changed }) => store.channels.delete(changed),
  };

const CALCULATION_STATUS_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof channel.statusZ
> = {
  channel: channel.CALCULATION_STATUS_CHANNEL_NAME,
  schema: channel.statusZ,
  onChange: async ({ store, changed, client }) =>
    store.channels.set(Number(changed.key), (p) => {
      if (p == null) return p;
      return client.channels.sugar({ ...p, status: changed });
    }),
};

export const useListenForCalculationStatus = (
  onChange: (status: channel.Status) => void,
): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(
    () =>
      store.channels.onSet((ch) => {
        if (ch.status == null) return;
        onChange(ch.status);
      }),
    [store],
  );
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  channel.Key,
  channel.Channel
> = {
  equal: (a, b) => deep.equal(a.payload, b.payload),
  listeners: [
    SET_CHANNEL_LISTENER,
    DELETE_CHANNEL_LISTENER,
    CALCULATION_STATUS_LISTENER,
  ],
};

export const formSchema = channel.newZ
  .extend({
    name: z.string().min(1, "Name must not be empty"),
    dataType: DataType.z.transform((v) => v.toString()),
    requires: channel.keyZ.array().optional(),
  })
  .refine(
    (v) => !v.isIndex || DataType.z.parse(v.dataType).equals(DataType.TIMESTAMP),
    {
      message: "Index channel must have data type TIMESTAMP",
      path: ["dataType"],
    },
  )
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual || v.expression !== "", {
    message: "Data channel must have an index",
    path: ["index"],
  })
  .refine((v) => v.virtual || !DataType.z.parse(v.dataType).isVariable, {
    message: "Persisted channels must have a fixed-size data type",
    path: ["dataType"],
  });

export const calculatedFormSchema = formSchema.safeExtend({
  expression: z
    .string()
    .min(1, "Expression must not be empty")
    .refine((v) => v.includes("return"), {
      message: "Expression must contain a return statement",
    }),
});

const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
});

export interface RetrieveQuery {
  key: channel.Key;
  rangeKey?: ranger.Key;
}

export const ZERO_FORM_VALUES: z.infer<
  typeof formSchema | typeof calculatedFormSchema
> = {
  key: 0,
  name: "",
  index: 0,
  dataType: DataType.FLOAT32.toString(),
  internal: false,
  isIndex: false,
  leaseholder: 0,
  virtual: false,
  expression: "",
  operations: [
    {
      type: "none",
      resetChannel: 0,
      duration: TimeSpan.ZERO,
    },
  ],
};

const retrieveSingle = async ({
  client,
  query: { key, rangeKey },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  let ch = store.channels.get(key);
  if (ch == null) {
    ch = await client.channels.retrieve(key);
    store.channels.set(ch.key, ch);
  }
  if (rangeKey != null) {
    const aliasKey = ranger.aliasKey({ range: rangeKey, channel: ch.key });
    let alias = store.rangeAliases.get(aliasKey);
    if (alias == null) {
      const aliasName = await client.ranges.retrieveAlias(rangeKey, ch.key);
      alias = { alias: aliasName, channel: ch.key, range: rangeKey };
      store.rangeAliases.set(aliasKey, alias);
    }
    if (alias != null) ch.alias = alias.alias;
  }
  return ch;
};

const retrieveMultiple = async ({
  client,
  query: { keys, rangeKey },
  store,
}: Flux.RetrieveParams<RetrieveMultipleQuery, FluxSubStore>) => {
  const channels = store.channels.get(keys);
  const existingKeys = new Set(channels?.map((ch) => ch.key));
  const missingKeys = keys.filter((key) => !existingKeys.has(key));
  if (missingKeys.length > 0) {
    const missingChannels = await client.channels.retrieve(missingKeys);
    channels.push(...missingChannels);
    store.channels.set(missingChannels);
  }
  if (rangeKey != null) {
    const aliasKeys = keys.map((key) =>
      ranger.aliasKey({ range: rangeKey, channel: key }),
    );
    const aliases = store.rangeAliases.get(aliasKeys);
    aliases.forEach((alias) => {
      if (alias == null) return;
      const ch = channels.find((ch) => ch.key === alias.channel);
      if (ch != null) ch.alias = alias.alias;
    });
    const existingAliasChannels = new Set(aliases.map((alias) => alias.channel));
    const missingAliasChannels = keys.filter((key) => !existingAliasChannels.has(key));
    if (missingAliasChannels.length > 0) {
      const missingAliases = await client.ranges.retrieveAliases(
        rangeKey,
        missingAliasChannels,
      );
      Object.entries(missingAliases).forEach(([channel, alias]: [string, string]) => {
        const chKey = Number(channel);
        const ch = channels.find((ch) => ch.key === chKey);
        if (ch != null) ch.alias = alias;
        const aliasKey = ranger.aliasKey({ range: rangeKey, channel: chKey });
        store.rangeAliases.set(aliasKey, { alias, channel: chKey, range: rangeKey });
      });
    }
  }
  return channels;
};

const retrieveInitialFormValues = async ({
  query: { key, rangeKey },
  store,
  client,
  reset,
}: Flux.FormRetrieveParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema,
  FluxSubStore
>) => {
  if (key == null) return undefined;
  const res = await retrieveSingle({ client, store, query: { key, rangeKey } });
  reset(channelToFormValues(res));
};

export const { useRetrieve, useRetrieveStateful, useRetrieveObservable } =
  Flux.createRetrieve<RetrieveQuery, channel.Channel, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: retrieveSingle,
    mountListeners: ({ store, onChange, query: { key, rangeKey }, client }) => {
      const ch = store.channels.onSet((channel) => {
        if (rangeKey != null) {
          const alias = store.rangeAliases.get(
            ranger.aliasKey({ range: rangeKey, channel: key }),
          );
          if (alias != null) channel.alias = alias.alias;
        }
        onChange(channel);
      }, key);
      if (rangeKey == null) return ch;
      const aliasKey = ranger.aliasKey({ range: rangeKey, channel: key });
      const onSetAlias = store.rangeAliases.onSet((alias) => {
        if (alias == null) return;
        onChange(
          state.skipUndefined((p) =>
            client.channels.sugar({ ...p, alias: alias.alias }),
          ),
        );
      }, aliasKey);
      const onDeleteAlias = store.rangeAliases.onDelete(
        () =>
          onChange(
            state.skipUndefined((p) =>
              client.channels.sugar({ ...p, alias: undefined }),
            ),
          ),
        aliasKey,
      );
      return [ch, onSetAlias, onDeleteAlias];
    },
  });

export interface RetrieveMultipleQuery extends channel.RetrieveOptions {
  keys: channel.Keys;
}

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  channel.Channel[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: retrieveMultiple,
  mountListeners: ({ store, onChange, query: { keys, rangeKey }, client }) => {
    const keysSet = new Set(keys);
    const ch = store.channels.onSet(async (channel) => {
      if (!keysSet.has(channel.key)) return;
      if (rangeKey != null) {
        const aliasKey = ranger.aliasKey({ range: rangeKey, channel: channel.key });
        let alias = store.rangeAliases.get(aliasKey);
        if (alias == null)
          try {
            const aliasName = await client.ranges.retrieveAlias(rangeKey, channel.key);
            alias = { alias: aliasName, channel: channel.key, range: rangeKey };
            store.rangeAliases.set(aliasKey, alias);
          } catch (e) {
            console.error(e);
          }

        if (alias != null) channel.alias = alias.alias;
      }
      onChange(
        state.skipUndefined((p) =>
          p.map((ch) => (ch.key === channel.key ? channel : ch)),
        ),
      );
    });
    if (rangeKey == null) return ch;
    const onSetAlias = store.rangeAliases.onSet((alias) => {
      if (alias == null) return;
      onChange(
        state.skipUndefined((p) =>
          p.map((ch) =>
            ch.key === alias.channel
              ? client.channels.sugar({ ...ch, alias: alias.alias })
              : ch,
          ),
        ),
      );
    });
    const onRemoveAlias = store.rangeAliases.onDelete((aliasKey) => {
      const decoded = ranger.decodeDeleteAliasChange(aliasKey);
      onChange(
        state.skipUndefined((p) =>
          p.map((ch) =>
            ch.key === decoded.channel
              ? client.channels.sugar({ ...ch, alias: undefined })
              : ch,
          ),
        ),
      );
    });
    return [ch, onSetAlias, onRemoveAlias];
  },
});

const updateForm = async ({
  client,
  store,
  set,
  value,
}: Flux.FormUpdateParams<
  typeof formSchema | typeof calculatedFormSchema,
  FluxSubStore
>) => {
  const values = value();
  if (values.requires != null) delete values.requires;
  const ch = await client.channels.create(value());
  store.channels.set(ch.key, ch);
  set("key", ch.key);
};

export interface FormQuery extends optional.Optional<RetrieveQuery, "key"> {}

const formMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema,
  FluxSubStore
>["mountListeners"] = ({ store, get, reset }) =>
  store.channels.onSet((changed) => {
    const key = get<channel.Key>("key").value;
    if (key !== changed.key) return;
    reset(channelToFormValues(changed));
  });

export const useForm = Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  update: updateForm,
  mountListeners: formMountListeners,
});

export const useCalculatedForm = Flux.createForm<
  FormQuery,
  typeof calculatedFormSchema,
  FluxSubStore
>({
  name: "calculated channel",
  schema: calculatedFormSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  update: updateForm,
  mountListeners: formMountListeners,
});

export interface ListQuery extends channel.RetrieveOptions {
  searchTerm?: string;
  rangeKey?: string;
  internal?: boolean;
  offset?: number;
  limit?: number;
}

const DEFAULT_LIST_PARAMS: ListQuery = {
  internal: false,
};

export const useList = Flux.createList<
  ListQuery,
  channel.Key,
  channel.Channel,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ query, store }) => {
    if (query.searchTerm != null && query.searchTerm.length > 0) return [];
    return store.channels.get((ch) => {
      if (query.internal != null && ch.internal !== query.internal) return false;
      if (
        primitive.isNonZero(query.notDataTypes) &&
        query.notDataTypes.some((dt) => new DataType(dt).equals(ch.dataType))
      )
        return false;
      if (
        primitive.isNonZero(query.dataTypes) &&
        !query.dataTypes.some((dt) => new DataType(dt).equals(ch.dataType))
      )
        return false;
      if (query.isIndex != null && ch.isIndex !== query.isIndex) return false;
      if (query.virtual != null && ch.virtual !== query.virtual) return false;
      return true;
    });
  },
  retrieve: async ({ client, query, store }) => {
    const channels = await client.channels.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...query,
    });
    store.channels.set(channels);
    return channels;
  },
  retrieveByKey: async ({ client, key, query, store }) =>
    await retrieveSingle({ client, query: { ...query, key }, store }),
  mountListeners: ({ store, onChange, onDelete, query: { rangeKey }, client }) => {
    const destructors = [
      store.channels.onSet(onChange),
      store.channels.onDelete(onDelete),
    ];
    if (rangeKey != null)
      destructors.push(
        store.rangeAliases.onSet((alias) => {
          if (alias.range !== rangeKey) return;
          onChange(alias.channel, (prev) => {
            if (prev == null) return prev;
            return client.channels.sugar({ ...prev, alias: alias.alias });
          });
        }),
      );
    return destructors;
  },
});

export interface RenameParams extends Pick<channel.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, name } = data;
    rollbacks.push(
      store.channels.set(
        key,
        state.skipUndefined((p) => client.channels.sugar({ ...p, name })),
      ),
    );
    rollbacks.push(Ontology.renameFluxResource(store, channel.ontologyID(key), name));
    await client.channels.rename(key, name);
    return data;
  },
});

const ALIAS_RESOURCE_NAME = "channel alias";

export interface UpdateAliasParams
  extends optional.Optional<ranger.Alias, "range" | "channel"> {
  alias: string;
}

export const { useUpdate: useUpdateAlias } = Flux.createUpdate<
  UpdateAliasParams,
  FluxSubStore
>({
  name: ALIAS_RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data: v, store }) => {
    const { range, channel, alias } = v;
    if (range == null || channel == null) return false;
    await client.ranges.setAlias(range, channel, alias);
    store.rangeAliases.set(ranger.aliasKey({ range, channel }), {
      channel,
      range,
      alias,
    });
    return v;
  },
});

export type DeleteParams = channel.Key | channel.Keys;

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = channel.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.channels.delete(keys));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    store.channels.delete(keys);
    await client.channels.delete(keys);
    return data;
  },
});

export interface DeleteAliasParams {
  range?: ranger.Key;
  channels?: channel.Key | channel.Key[];
}

export const { useUpdate: useDeleteAlias } = Flux.createUpdate<
  DeleteAliasParams,
  FluxSubStore,
  DeleteAliasParams
>({
  name: ALIAS_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, store, data, rollbacks }) => {
    const { range, channels } = data;
    if (range == null || channels == null) return false;
    const arrChannels = array.toArray(channels);
    await client.ranges.deleteAlias(range, arrChannels);
    const aliasKeys = arrChannels.map((c) => ranger.aliasKey({ range, channel: c }));
    rollbacks.push(store.rangeAliases.delete(aliasKeys));
    return data;
  },
});

interface RetrieveGroupQuery {}

export const { useRetrieve: useRetrieveGroup } = Flux.createRetrieve<
  RetrieveGroupQuery,
  group.Group,
  FluxSubStore
>({
  name: "Channel Group",
  retrieve: async ({ client, store }) => {
    const g = await client.channels.retrieveGroup();
    store.groups.set(g.key, g);
    return g;
  },
});
