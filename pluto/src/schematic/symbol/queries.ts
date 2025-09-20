// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, ontology, schematic } from "@synnaxlabs/client";

import { Flux } from "@/flux";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "schematicSymbols";

const RESOURCE_NAME = "Schematic Symbol";

export interface FluxStore
  extends Flux.UnaryStore<schematic.symbol.Key, schematic.symbol.Symbol> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
}

const SET_SYMBOL_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof schematic.symbol.symbolZ
> = {
  channel: schematic.symbol.SET_CHANNEL_NAME,
  schema: schematic.symbol.symbolZ,
  onChange: ({ store, changed }) => store.schematicSymbols.set(changed.key, changed),
};

const DELETE_SYMBOL_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof schematic.symbol.keyZ
> = {
  channel: schematic.symbol.DELETE_CHANNEL_NAME,
  schema: schematic.symbol.keyZ,
  onChange: ({ store, changed }) => store.schematicSymbols.delete(changed),
};

export const STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  schematic.symbol.Key,
  schematic.symbol.Symbol
> = {
  listeners: [SET_SYMBOL_LISTENER, DELETE_SYMBOL_LISTENER],
};

export interface RetrieveQuery {
  key: string;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.schematicSymbols.get(key);
  if (cached != null) return cached;
  const symbol = await client.workspaces.schematics.symbols.retrieve({ key });
  store.schematicSymbols.set(key, symbol);
  return symbol;
};

export const { useRetrieve, useRetrieveEffect } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.symbol.Symbol,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.schematicSymbols.onSet(onChange, key),
  ],
});

export interface ListQuery {
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
  ListQuery,
  string,
  schematic.symbol.Symbol,
  FluxSubStore
>({
  sort: (a, b) => a.name.localeCompare(b.name),
  retrieveCached: ({ query: params, store }) => {
    if (params.searchTerm != null && params.searchTerm.length > 0) return [];
    if (params.parent == null) return store.schematicSymbols.list();
    const keys = store.relationships.get((r) =>
      matchSymbolRelationship(r, params.parent as ontology.ID),
    );
    return store.schematicSymbols.get(keys.map((k) => k.to.key));
  },
  name: "Schematic Symbols",
  retrieve: async ({ client, store, query: { parent, ...rest } }) => {
    if (parent != null) {
      const children = await client.ontology.retrieveChildren(parent);
      const keys = children.map((c) => c.id.key);
      if (keys.length === 0) return [];
      const symbols = await client.workspaces.schematics.symbols.retrieve({
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
    const res = await client.workspaces.schematics.symbols.retrieve(rest);
    store.schematicSymbols.set(res);
    return res;
  },
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete, query, client }) => [
    store.schematicSymbols.onSet((symbol) =>
      onChange(symbol.key, (p) => (p == null ? null : symbol)),
    ),
    store.schematicSymbols.onDelete(onDelete),
    store.relationships.onSet(async (r) => {
      if (!matchSymbolRelationship(r, query.parent as ontology.ID)) return;
      const symbol = await retrieveSingle({ client, query: { key: r.to.key }, store });
      onChange(r.to.key, symbol);
    }),
    store.relationships.onDelete(async (r) => {
      const rel = ontology.relationshipZ.parse(r);
      if (!matchSymbolRelationship(rel, query.parent as ontology.ID)) return;
      onDelete(rel.to.key);
    }),
  ],
});

export interface FormQuery {
  key?: string;
  parent?: ontology.ID;
}

export const formSchema = schematic.symbol.symbolZ
  .partial({ key: true })
  .extend({ parent: ontology.idZ });

export const useForm = Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
  name: RESOURCE_NAME,
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
  retrieve: async ({ client, query: { key, parent }, reset, store }) => {
    if (key == null) return;
    const symbol = await retrieveSingle({ client, store, query: { key } });
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
    const created = await client.workspaces.schematics.symbols.create(payload);
    reset({ ...created, parent: payload.parent });
  },
  mountListeners: ({ store, query: { parent, key }, reset, get }) => {
    if (key == null) return [];
    return [
      store.schematicSymbols.onSet(
        (symbol) =>
          reset({
            ...symbol,
            parent:
              parent ??
              get<ontology.ID>("parent", { optional: true })?.value ??
              ontology.ROOT_ID,
          }),
        key,
      ),
    ];
  },
});

export interface RenameParams {
  key: string;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: "rename",
  update: async ({ client, data, store }) => {
    await client.workspaces.schematics.symbols.rename(value.key, value.name);
    store.schematicSymbols.set(value.key, (p) => {
      if (p == null) return p;
      return { ...p, name: value.name };
    });
    return value;
  },
});

export interface DeleteArgs {
  key: string;
}

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: "delete",
  update: async ({ client, data, store }) => {
    await client.workspaces.schematics.symbols.delete(value.key);
    store.schematicSymbols.delete(value.key);
    return value;
  },
});

export const { useRetrieve: useRetrieveGroup } = Flux.createRetrieve<
  {},
  group.Group,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client }) =>
    await client.workspaces.schematics.symbols.retrieveGroup(),
});
