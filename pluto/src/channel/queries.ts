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
import { z } from "zod";

import { Flux } from "@/flux";
import { type Ranger } from "@/ranger";

export interface FluxStore extends Flux.UnaryStore<channel.Key, channel.Channel> {}

interface SubStore extends Flux.Store {
  channels: FluxStore;
  rangeAliases: Ranger.AliasFluxStore;
}

const SET_CHANNEL_LISTENER: Flux.ChannelListener<SubStore, typeof channel.keyZ> = {
  channel: channel.SET_CHANNEL_NAME,
  schema: channel.keyZ,
  onChange: async ({ store, changed, client }) => {
    const ch = await client.channels.retrieve(changed);
    store.channels.set(changed, ch);
  },
};

const DELETE_CHANNEL_LISTENER: Flux.ChannelListener<SubStore, typeof channel.keyZ> = {
  channel: channel.DELETE_CHANNEL_NAME,
  schema: channel.keyZ,
  onChange: async ({ store, changed }) => store.channels.delete(changed),
};

const CALCULATION_STATUS_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof channel.statusZ
> = {
  channel: channel.CALCULATION_STATUS_CHANNEL_NAME,
  schema: channel.statusZ,
  onChange: async ({ store, changed, client }) => {
    store.channels.set(Number(changed.key), (p) => {
      if (p == null) return p;
      return client.channels.sugar({ ...p, status: changed });
    });
  },
};

export const useListenForCalculationStatus = (
  onChange: (status: channel.Status) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  store.channels.onSet(async (ch) => {
    if (ch.status == null) return;
    onChange(ch.status);
  });
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
  mountListeners: ({ store, onChange, params: { key }, client }) => [
    store.channels.onSet(async (channel) => onChange(channel), key),
    store.rangeAliases.onSet(async (alias) => {
      if (alias.channel !== key) return;
      onChange((p) => client.channels.sugar({ ...p, alias: alias.alias }));
    }),
  ],
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

export const useForm = (args: Flux.UseFormArgs<FormRetrieveArgs, typeof formSchema>) =>
  Flux.createForm<FormRetrieveArgs, typeof formSchema, SubStore>({
    name: "Channel",
    schema: formSchema,
    initialValues: ZERO_FORM_VALUES,
    retrieve: formRetrieveFn,
    update: updateForm,
    mountListeners: ({ store, params, onChange }) => [
      store.channels.onSet(async (channel) => {
        if (params.key !== channel.key) return;
        onChange(channelToFormValues(channel));
      }, params.key),
    ],
  })(args);

export const useCalculatedForm = (
  args: Flux.UseFormArgs<FormRetrieveArgs, typeof calculatedFormSchema>,
) =>
  Flux.createForm<FormRetrieveArgs, typeof calculatedFormSchema, SubStore>({
    name: "Calculated Channel",
    schema: calculatedFormSchema,
    initialValues: ZERO_FORM_VALUES,
    retrieve: formRetrieveFn,
    update: updateForm,
    mountListeners: ({ store, params, onChange }) => [
      store.channels.onSet(async (channel) => {
        if (params.key !== channel.key) return;
        onChange(channelToFormValues(channel));
      }, params.key),
    ],
  })(args);

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
    const results = await client.channels.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
      searchTerm: params.searchTerm,
    });
    results.forEach((ch) => store.channels.set(ch.key, ch, { notify: false }));
    return results;
  },
  retrieveByKey: async ({ client, key }) => await client.channels.retrieve(key),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.channels.onSet(async (channel) => {
      onChange(channel.key, channel);
    }),
    store.channels.onDelete(async (key) => onDelete(key)),
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

interface RetrieveAliasArgs extends Optional<UpdateArgs, "key"> {
  rangeKey?: string;
  channelKey: channel.Key;
}

export const retrieveAlias = Flux.createRetrieve<
  RetrieveAliasArgs,
  ranger.Alias | undefined,
  SubStore
>({
  name: "Channel Alias",
  retrieve: async ({ client, params: { rangeKey, channelKey }, store }) => {
    if (rangeKey == null) return undefined;
    let alias = store.rangeAliases.get(
      ranger.aliasKey({ range: rangeKey, channel: channelKey }),
    );
    if (alias == null) {
      const aliasName = await client.ranges.retrieveAlias(rangeKey, channelKey);
      alias = { alias: aliasName, channel: channelKey, range: rangeKey };
      store.rangeAliases.set(
        ranger.aliasKey({ range: rangeKey, channel: channelKey }),
        alias,
      );
    }
    return alias;
  },
  mountListeners: ({ store, onChange, params: { channelKey } }) => [
    store.rangeAliases.onSet(async (alias) => {
      if (alias.channel !== channelKey) return;
      onChange(alias);
    }),
    store.rangeAliases.onDelete(async (changedKey) => {
      onChange((p) => {
        if (p == null) return p;
        const key = ranger.aliasKey(p);
        if (key !== changedKey) return p;
        return undefined;
      });
    }),
  ],
});

export interface UseNameReturn {
  name: string;
  alias?: string;
  rename: (name: string) => void;
}

interface UseNameArgs {
  key: channel.Key;
  range?: ranger.Key;
  defaultName: string;
}

export const useName = ({ key, range, defaultName }: UseNameArgs): UseNameReturn => {
  const args = { params: { key, rangeKey: range } };
  const { data } = retrieve.useDirect(args);
  const { update: rename } = update.useDirect(args);
  const handleRename = (name: string) => {
    if (data == null) return;
    rename({ ...data, name });
  };
  const name = data?.alias ?? data?.name ?? defaultName;
  const alias = data?.alias;
  return { name, alias, rename: handleRename };
};
