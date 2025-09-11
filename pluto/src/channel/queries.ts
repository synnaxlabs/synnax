// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, ranger } from "@synnaxlabs/client";
import { deep, type Optional, primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { type Ranger } from "@/ranger";

export const FLUX_STORE_KEY = "channels";

export interface FluxStore extends Flux.UnaryStore<channel.Key, channel.Channel> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.AliasFluxStore;
}

const SET_CHANNEL_LISTENER: Flux.ChannelListener<SubStore, typeof channel.keyZ> = {
  channel: channel.SET_CHANNEL_NAME,
  schema: channel.keyZ,
  onChange: async ({ store, changed, client }) =>
    store.channels.set(changed, await client.channels.retrieve(changed)),
};

const DELETE_CHANNEL_LISTENER: Flux.ChannelListener<SubStore, typeof channel.keyZ> = {
  channel: channel.DELETE_CHANNEL_NAME,
  schema: channel.keyZ,
  onChange: ({ store, changed }) => store.channels.delete(changed),
};

const CALCULATION_STATUS_LISTENER: Flux.ChannelListener<
  SubStore,
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
  const store = Flux.useStore<SubStore>();
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
  SubStore,
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
  })
  .refine(
    (v) => !v.isIndex || DataType.z.parse(v.dataType).equals(DataType.TIMESTAMP),
    {
      message: "Index channel must have data type TIMESTAMP",
      path: ["dataType"],
    },
  )
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual, {
    message: "Data channel must have an index",
    path: ["index"],
  })
  .refine((v) => v.virtual || !DataType.z.parse(v.dataType).isVariable, {
    message: "Persisted channels must have a fixed-size data type",
    path: ["dataType"],
  });

export const calculatedFormSchema = formSchema
  .extend({
    expression: z
      .string()
      .min(1, "Expression must not be empty")
      .refine((v) => v.includes("return"), {
        message: "Expression must contain a return statement",
      }),
  })
  .refine((v) => v.requires?.length > 0, {
    message: "Expression must use at least one channel",
    path: ["requires"],
  });

const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
});

export interface RetrieveArgs {
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
  requires: [],
};

const retrieveSingleFn = async ({
  client,
  params: { key, rangeKey },
  store,
}: Flux.RetrieveArgs<RetrieveArgs, SubStore>) => {
  let ch = store.channels.get(key);
  if (ch == null) {
    ch = await client.channels.retrieve(key);
    store.channels.set(ch.key, ch);
  }
  if (rangeKey != null) {
    const aliasKey = ranger.aliasKey({ range: rangeKey, channel: ch.key });
    let alias = store.rangeAliases.get(aliasKey);
    if (alias == null)
      try {
        const aliasName = await client.ranges.retrieveAlias(rangeKey, ch.key);
        alias = { alias: aliasName, channel: ch.key, range: rangeKey };
      } finally {
        store.rangeAliases.set(aliasKey, { channel: ch.key, range: rangeKey });
      }

    if (alias != null) ch.alias = alias.alias;
  }
  return ch;
};

const retrieveManyFn = async ({
  client,
  params: { keys, rangeKey },
  store,
}: Flux.RetrieveArgs<RetrieveManyArgs, SubStore>) => {
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

const formRetrieveFn = async ({
  params: { key, rangeKey },
  store,
  client,
  reset,
}: Flux.FormRetrieveArgs<
  FormRetrieveArgs,
  typeof formSchema | typeof calculatedFormSchema,
  SubStore
>) => {
  if (key == null) return undefined;
  const res = await retrieveSingleFn({ client, store, params: { key, rangeKey } });
  reset(channelToFormValues(res));
};

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveArgs,
  channel.Channel,
  SubStore
>({
  name: "Channel",
  retrieve: retrieveSingleFn,
  mountListeners: ({ store, onChange, params: { key, rangeKey }, client }) => {
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
      onChange((p) => client.channels.sugar({ ...p, alias: alias.alias }));
    }, aliasKey);
    const onDeleteAlias = store.rangeAliases.onDelete(
      () => onChange((p) => client.channels.sugar({ ...p, alias: undefined })),
      aliasKey,
    );
    return [ch, onSetAlias, onDeleteAlias];
  },
});

export interface RetrieveManyArgs extends channel.RetrieveOptions {
  keys: channel.Keys;
}

export const { useRetrieve: useRetrieveMany } = Flux.createRetrieve<
  RetrieveManyArgs,
  channel.Channel[],
  SubStore
>({
  name: "Channels",
  retrieve: retrieveManyFn,
  mountListeners: ({ store, onChange, params: { keys, rangeKey }, client }) => {
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
      onChange((p) => p.map((ch) => (ch.key === channel.key ? channel : ch)));
    });
    if (rangeKey == null) return ch;
    const onSetAlias = store.rangeAliases.onSet((alias) => {
      if (alias == null) return;
      onChange((p) =>
        p.map((ch) =>
          ch.key === alias.channel
            ? client.channels.sugar({ ...ch, alias: alias.alias })
            : ch,
        ),
      );
    });
    const onRemoveAlias = store.rangeAliases.onDelete((aliasKey) => {
      const decoded = ranger.decodeDeleteAliasChange(aliasKey);
      onChange((p) =>
        p.map((ch) =>
          ch.key === decoded.channel
            ? client.channels.sugar({ ...ch, alias: undefined })
            : ch,
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
}: Flux.FormUpdateArgs<typeof formSchema | typeof calculatedFormSchema, SubStore>) => {
  const ch = await client.channels.create(value());
  store.channels.set(ch.key, ch);
  set("key", ch.key);
};

export interface FormRetrieveArgs extends Optional<RetrieveArgs, "key"> {}

const formMountListeners: Flux.CreateFormArgs<
  FormRetrieveArgs,
  typeof formSchema | typeof calculatedFormSchema,
  SubStore
>["mountListeners"] = ({ store, get, reset }) =>
  store.channels.onSet((changed) => {
    const key = get<channel.Key>("key").value;
    if (key !== changed.key) return;
    reset(channelToFormValues(changed));
  });

export const useForm = Flux.createForm<FormRetrieveArgs, typeof formSchema, SubStore>({
  name: "Channel",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: formRetrieveFn,
  update: updateForm,
  mountListeners: formMountListeners,
});

export const useCalculatedForm = Flux.createForm<
  FormRetrieveArgs,
  typeof calculatedFormSchema,
  SubStore
>({
  name: "Calculated Channel",
  schema: calculatedFormSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: formRetrieveFn,
  update: updateForm,
  mountListeners: formMountListeners,
});

export interface ListParams extends channel.RetrieveOptions {
  searchTerm?: string;
  rangeKey?: string;
  internal?: boolean;
  offset?: number;
  limit?: number;
}

const DEFAULT_LIST_PARAMS: ListParams = {
  internal: false,
};

export const useList = Flux.createList<
  ListParams,
  channel.Key,
  channel.Channel,
  SubStore
>({
  name: "Channels",
  retrieveCached: ({ params, store }) => {
    if (params.searchTerm != null && params.searchTerm.length > 0) return [];
    return store.channels.get((ch) => {
      if (params.internal != null && ch.internal !== params.internal) return false;
      if (params.calculated != null && ch.isCalculated !== params.calculated)
        return false;
      if (
        primitive.isNonZero(params.notDataTypes) &&
        params.notDataTypes.some((dt) => new DataType(dt).equals(ch.dataType))
      )
        return false;
      if (
        primitive.isNonZero(params.dataTypes) &&
        !params.dataTypes.some((dt) => new DataType(dt).equals(ch.dataType))
      )
        return false;
      if (params.isIndex != null && ch.isIndex !== params.isIndex) return false;
      if (params.virtual != null && ch.virtual !== params.virtual) return false;
      return true;
    });
  },
  retrieve: async ({ client, params, store }) => {
    const channels = await client.channels.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
    });
    store.channels.set(channels);
    return channels;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveSingleFn({ client, params: { key }, store }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.channels.onSet((channel) => onChange(channel.key, channel)),
    store.channels.onDelete((key) => onDelete(key)),
  ],
});

export const update = Flux.createUpdate<channel.New, SubStore>({
  name: "Channel",
  update: async ({ client, value, store }) => {
    const ch = await client.channels.create(value);
    store.channels.set(ch.key, ch);
  },
});

interface RenameArgs {
  key: channel.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameArgs, SubStore>({
  name: "Channel",
  update: async ({ client, value, store }) => {
    const { key, name } = value;
    if (key == null) return;
    await client.channels.rename(key, name);
    store.channels.set(key, (p) => {
      if (p == null) return p;
      return client.channels.sugar({ ...p, name });
    });
  },
});

interface UpdateAliasArgs extends Optional<ranger.Alias, "range" | "channel"> {
  alias: string;
}

export const { useUpdate: useUpdateAlias } = Flux.createUpdate<
  UpdateAliasArgs,
  SubStore
>({
  name: "Channel Alias",
  update: async ({ client, value: v, store }) => {
    const { range, channel, alias } = v;
    if (range == null || channel == null) return;
    await client.ranges.setAlias(range, channel, alias);
    store.rangeAliases.set(ranger.aliasKey({ range, channel }), {
      channel,
      range,
      alias,
    });
  },
});
