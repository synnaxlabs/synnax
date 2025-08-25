// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, ontology, schematic, type Synnax } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "schematicSymbols";

export interface FluxStore
  extends Flux.UnaryStore<schematic.symbol.Key, schematic.symbol.Symbol> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_SYMBOL_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof schematic.symbol.symbolZ
> = {
  channel: schematic.symbol.SET_CHANNEL_NAME,
  schema: schematic.symbol.symbolZ,
  onChange: ({ store, changed }) => store.schematicSymbols.set(changed.key, changed),
};

const DELETE_SYMBOL_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof schematic.symbol.keyZ
> = {
  channel: schematic.symbol.DELETE_CHANNEL_NAME,
  schema: schematic.symbol.keyZ,
  onChange: ({ store, changed }) => store.schematicSymbols.delete(changed),
};

export const STORE_CONFIG: Flux.UnaryStoreConfig<
  SubStore,
  schematic.symbol.Key,
  schematic.symbol.Symbol
> = {
  listeners: [SET_SYMBOL_LISTENER, DELETE_SYMBOL_LISTENER],
};

export interface RetrieveParams {
  key: string;
}

const retrieveByKey = async (client: Synnax, key: string, store: SubStore) => {
  const cached = store.schematicSymbols.get(key);
  if (cached != null) return cached;
  const symbol = await client.workspaces.schematic.symbols.retrieve({ key });
  store.schematicSymbols.set(key, symbol);
  return symbol;
};

export const retrieve = Flux.createRetrieve<
  RetrieveParams,
  schematic.symbol.Symbol,
  SubStore
>({
  name: "SchematicSymbols",
  retrieve: async ({ client, params, store }) =>
    await retrieveByKey(client, params.key, store),
  mountListeners: ({ store, params, onChange }) => [
    store.schematicSymbols.onSet(onChange, params.key),
  ],
});

export interface ListParams {
  keys?: string[];
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<
  ListParams,
  string,
  schematic.symbol.Symbol,
  SubStore
>({
  name: "Schematic Symbols",
  retrieve: async ({ client, params: { parent, ...rest } }) => {
    if (parent != null) {
      const children = await client.ontology.retrieveChildren(parent);
      const keys = children.map((c) => c.id.key);
      return await client.workspaces.schematic.symbols.retrieve({
        ...rest,
        keys,
      });
    }
    return await client.workspaces.schematic.symbols.retrieve(rest);
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveByKey(client, key, store),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.schematicSymbols.onSet((symbol) => onChange(symbol.key, symbol)),
    store.schematicSymbols.onDelete(onDelete),
  ],
});

export interface UseFormParams {
  key?: string;
  parent?: ontology.ID;
}

export const formSchema = schematic.symbol.symbolZ
  .partial({
    key: true,
  })
  .extend({
    parent: ontology.idZ,
  });

export const useForm = Flux.createForm<UseFormParams, typeof formSchema, SubStore>({
  name: "SchematicSymbols",
  initialValues: {
    name: "",
    data: { svg: "", states: [], handles: [], variant: "static" },
    parent: ontology.ROOT_ID,
  },
  schema: formSchema,
  retrieve: async ({ client, params: { key, parent }, reset, store }) => {
    if (key == null) return;
    const symbol = await retrieveByKey(client, key, store);
    reset({
      name: symbol.name,
      data: symbol.data,
      key: symbol.key,
      parent: parent ?? ontology.ROOT_ID,
    });
  },
  update: async ({ client, value, reset, params }) => {
    const created = await client.workspaces.schematic.symbols.create({
      ...value(),
      parent: params.parent ?? ontology.ROOT_ID,
    });
    reset({ ...created, parent: params.parent ?? ontology.ROOT_ID });
  },
  mountListeners: ({ store, params, reset }) => {
    if (params.key == null) return [];
    return [
      store.schematicSymbols.onSet(
        (symbol) => reset({ ...symbol, parent: params.parent ?? ontology.ROOT_ID }),
        params.key,
      ),
    ];
  },
});

export interface RenameParams {
  key: string;
}

export const useRename = Flux.createUpdate<RenameParams, string>({
  name: "SchematicSymbols",
  update: async ({ client, value, params }) =>
    await client.workspaces.schematic.symbols.rename(params.key, value),
}).useDirect;

export interface DeleteParams {
  key: string;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "SchematicSymbols",
  update: async ({ client, params: { key } }) =>
    await client.workspaces.schematic.symbols.delete(key),
}).useDirect;

export const useGroup = Flux.createRetrieve<{}, group.Payload, SubStore>({
  name: "SchematicSymbols",
  retrieve: async ({ client }) =>
    await client.workspaces.schematic.symbols.retrieveGroup(),
});
