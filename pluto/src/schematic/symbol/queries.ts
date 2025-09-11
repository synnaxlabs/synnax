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
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "schematicSymbols";

export interface FluxStore
  extends Flux.UnaryStore<schematic.symbol.Key, schematic.symbol.Symbol> {}

export interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
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

export const { useRetrieve, useRetrieveEffect } = Flux.createRetrieve<
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

const matchSymbolRelationship = (r: ontology.Relationship, parent: ontology.ID) =>
  ontology.matchRelationship(r, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "schematic_symbol" },
  });
export const useList = Flux.createList<
  ListParams,
  string,
  schematic.symbol.Symbol,
  SubStore
>({
  sort: (a, b) => a.name.localeCompare(b.name),
  retrieveCached: ({ params, store }) => {
    if (params.searchTerm != null && params.searchTerm.length > 0) return [];
    if (params.parent == null) return store.schematicSymbols.list();
    const keys = store.relationships.get((r) =>
      matchSymbolRelationship(r, params.parent as ontology.ID),
    );
    return store.schematicSymbols.get(keys.map((k) => k.to.key));
  },
  name: "Schematic Symbols",
  retrieve: async ({ client, store, params: { parent, ...rest } }) => {
    if (parent != null) {
      const children = await client.ontology.retrieveChildren(parent);
      const keys = children.map((c) => c.id.key);
      if (keys.length === 0) return [];
      const symbols = await client.workspaces.schematic.symbols.retrieve({
        ...rest,
        keys,
      });
      symbols.forEach((s) => {
        const rel = {
          from: parent,
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: schematic.symbol.ontologyID(s.key),
        };
        store.relationships.set(ontology.relationshipToString(rel), rel);
      });
      return symbols;
    }
    const res = await client.workspaces.schematic.symbols.retrieve(rest);
    store.schematicSymbols.set(res);
    return res;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveByKey(client, key, store),
  mountListeners: ({ store, onChange, onDelete, params, client }) => [
    store.schematicSymbols.onSet((symbol) =>
      onChange(symbol.key, (p) => (p == null ? null : symbol)),
    ),
    store.schematicSymbols.onDelete(onDelete),
    store.relationships.onSet(async (r) => {
      if (!matchSymbolRelationship(r, params.parent as ontology.ID)) return;
      const symbol = await retrieveByKey(client, r.to.key, store);
      onChange(r.to.key, symbol);
    }),
    store.relationships.onDelete(async (r) => {
      const rel = ontology.relationshipZ.parse(r);
      if (!matchSymbolRelationship(rel, params.parent as ontology.ID)) return;
      onDelete(rel.to.key);
    }),
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
    version: 1,
    name: "",
    data: {
      svg: "",
      states: [],
      handles: [],
      variant: "static",
      scale: 1,
      scaleStroke: false,
      previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
    },
    parent: ontology.ROOT_ID,
  },
  schema: formSchema,
  retrieve: async ({ client, params: { key, parent }, reset, store }) => {
    if (key == null) return;
    const symbol = await retrieveByKey(client, key, store);
    if (parent == null) {
      const parents = await client.ontology.retrieveParents(
        schematic.symbol.ontologyID(key),
      );
      parent = parents[0].id;
    }
    reset({
      version: 1,
      name: symbol.name,
      data: symbol.data,
      key: symbol.key,
      parent,
    });
  },
  update: async ({ client, value, reset }) => {
    const payload = value();
    const created = await client.workspaces.schematic.symbols.create(payload);
    reset({ ...created, parent: payload.parent });
  },
  mountListeners: ({ store, params, reset, get }) => {
    if (params.key == null) return [];
    return [
      store.schematicSymbols.onSet(
        (symbol) =>
          reset({
            ...symbol,
            parent:
              params.parent ??
              get<ontology.ID>("parent", { optional: true })?.value ??
              ontology.ROOT_ID,
          }),
        params.key,
      ),
    ];
  },
});

export interface RenameArgs {
  key: string;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameArgs, SubStore>({
  name: "SchematicSymbols",
  update: async ({ client, value, store }) => {
    await client.workspaces.schematic.symbols.rename(value.key, value.name);
    store.schematicSymbols.set(value.key, (p) => {
      if (p == null) return p;
      return { ...p, name: value.name };
    });
  },
});

export interface DeleteArgs {
  key: string;
}

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteArgs, SubStore>({
  name: "SchematicSymbols",
  update: async ({ client, value, store }) => {
    await client.workspaces.schematic.symbols.delete(value.key);
    store.schematicSymbols.delete(value.key);
  },
});

export const { useRetrieve: useRetrieveGroup } = Flux.createRetrieve<
  {},
  group.Payload,
  SubStore
>({
  name: "SchematicSymbols",
  retrieve: async ({ client }) =>
    await client.workspaces.schematic.symbols.retrieveGroup(),
});
