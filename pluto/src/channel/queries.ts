// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, ranger } from "@synnaxlabs/client";
import { type Optional } from "@synnaxlabs/x";
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

export const STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
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

const retrieveFn = async ({
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
    let alias = store.rangeAliases.get(
      ranger.aliasKey({ range: rangeKey, channel: ch.key }),
    );
    if (alias == null) {
      const aliasName = await client.ranges.retrieveAlias(rangeKey, ch.key);
      alias = { alias: aliasName, channel: ch.key, range: rangeKey };
      store.rangeAliases.set(
        ranger.aliasKey({ range: rangeKey, channel: ch.key }),
        alias,
      );
    }
    ch.alias = alias.alias;
  }
  return ch;
};

const formRetrieveFn = async (args: Flux.RetrieveArgs<FormRetrieveArgs, SubStore>) => {
  if (args.params.key == null) return null;
  return channelToFormValues(
    await retrieveFn({
      ...args,
      params: { key: args.params.key, rangeKey: args.params.rangeKey },
    }),
  );
};

export const retrieve = Flux.createRetrieve<RetrieveArgs, channel.Channel, SubStore>({
  name: "Channel",
  retrieve: retrieveFn,
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
    const onSetAlias = store.rangeAliases.onSet(
      ({ alias }) => onChange((p) => client.channels.sugar({ ...p, alias })),
      aliasKey,
    );
    const onDeleteAlias = store.rangeAliases.onDelete(
      () => onChange((p) => client.channels.sugar({ ...p, alias: undefined })),
      aliasKey,
    );
    return [ch, onSetAlias, onDeleteAlias];
  },
});

const updateForm = async ({
  client,
  value,
  onChange,
  store,
}: Flux.UpdateArgs<
  FormRetrieveArgs,
  z.infer<typeof formSchema | typeof calculatedFormSchema>,
  SubStore
>) => {
  const ch = await client.channels.create(value);
  store.channels.set(ch.key, ch);
  onChange(channelToFormValues(ch));
};

export interface FormRetrieveArgs extends Optional<RetrieveArgs, "key"> {}

const formMountListeners: Flux.CreateFormArgs<
  FormRetrieveArgs,
  typeof formSchema | typeof calculatedFormSchema,
  SubStore
>["mountListeners"] = ({ store, onChange }) =>
  store.channels.onSet((changed) => {
    onChange((p) => {
      if (p == null || p.key !== changed.key) return p;
      return channelToFormValues(changed);
    });
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
  retrieve: async ({ client, params, store }) => {
    const channels = await client.channels.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
    });
    store.channels.set(channels);
    return channels;
  },
  retrieveByKey: async ({ client, key }) => await client.channels.retrieve(key),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.channels.onSet((channel) => onChange(channel.key, channel)),
    store.channels.onDelete((key) => onDelete(key)),
  ],
});

export interface UpdateArgs extends Optional<RetrieveArgs, "key"> {}

export const update = Flux.createUpdate<UpdateArgs, channel.New, SubStore>({
  name: "Channel",
  update: async ({ client, value, store }) => {
    const ch = await client.channels.create(value);
    store.channels.set(ch.key, ch);
  },
});

export interface RenameArgs extends Optional<UpdateArgs, "key"> {}

export const rename = Flux.createUpdate<RenameArgs, string, SubStore>({
  name: "Channel",
  update: async ({ client, value, store, params: { key } }) => {
    if (key == null) return;
    await client.channels.rename(key, value);
    store.channels.set(key, (p) => {
      if (p == null) return p;
      return client.channels.sugar({ ...p, name: value });
    });
  },
});

interface UpdateAliasArgs extends Optional<UpdateArgs, "key"> {
  rangeKey?: string;
  channelKey: channel.Key;
}

export const updateAlias = Flux.createUpdate<UpdateAliasArgs, string, SubStore>({
  name: "Channel Alias",
  update: async ({ client, value, store, params: { rangeKey, channelKey } }) => {
    if (rangeKey == null) return;
    const alias: ranger.Alias = { alias: value, channel: channelKey, range: rangeKey };
    await client.ranges.setAlias(rangeKey, channelKey, value);
    store.rangeAliases.set(ranger.aliasKey(alias), alias);
  },
});
