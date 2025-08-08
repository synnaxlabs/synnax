// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, schematic } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface RetrieveParams {
  key: string;
}

export const retrieve = Flux.createRetrieve<RetrieveParams, schematic.symbol.Symbol>({
  name: "SchematicSymbols",
  retrieve: async ({ client, params }) =>
    await client.workspaces.schematic.symbols.retrieve({ key: params.key }),
  listeners: [
    {
      channel: schematic.symbol.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        schematic.symbol.symbolZ,
        async ({ changed, onChange, params }) =>
          params.key === changed.key && onChange(changed),
      ),
    },
  ],
});

export interface ListParams extends schematic.symbol.Spec {
  keys?: string[];
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, string, schematic.symbol.Symbol>({
  name: "SchematicSymbols",
  retrieve: async ({ client, params }) =>
    await client.workspaces.schematic.symbols.retrieve(params),
  retrieveByKey: async ({ client, key }) =>
    await client.workspaces.schematic.symbols.retrieve({ key }),
  listeners: [
    {
      channel: schematic.symbol.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        schematic.symbol.symbolZ,
        async ({ changed, onChange }) => onChange(changed.key, changed),
      ),
    },
    {
      channel: schematic.symbol.DELETE_CHANNEL_NAME,
      onChange: Flux.stringHandler(async ({ changed, onDelete }) => onDelete(changed)),
    },
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

export const useForm = Flux.createForm<UseFormParams, typeof formSchema>({
  name: "SchematicSymbols",
  initialValues: { name: "", data: { svg: "", states: [] }, parent: ontology.ROOT_ID },
  schema: formSchema,
  retrieve: async ({ client, params: { key, parent } }) => {
    if (key == null) return null;
    const symbol = await client.workspaces.schematic.symbols.retrieve({ key });
    return {
      name: symbol.name,
      data: symbol.data,
      key: symbol.key,
      parent: parent ?? ontology.ROOT_ID,
    };
  },
  update: async ({ client, value, onChange, params }) => {
    const created = await client.workspaces.schematic.symbols.create({
      ...value,
      parent: params.parent ?? ontology.ROOT_ID,
    });
    onChange({ ...created, parent: params.parent ?? ontology.ROOT_ID });
  },
  listeners: [
    {
      channel: schematic.symbol.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        schematic.symbol.symbolZ,
        async ({ changed, onChange, params }) =>
          (params.key == null || changed.key !== params.key) &&
          onChange({ ...changed, parent: params.parent ?? ontology.ROOT_ID }),
      ),
    },
  ],
});

export interface RenameParams {
  key: string;
  name: string;
}

export const useRename = Flux.createUpdate<RenameParams, void>({
  name: "SchematicSymbols",
  update: async ({ client, params }) =>
    await client.workspaces.schematic.symbols.rename(params.key, params.name),
}).useDirect;

export interface DeleteParams {
  key: string;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "SchematicSymbols",
  update: async ({ client, params: { key } }) =>
    await client.workspaces.schematic.symbols.delete(key),
}).useDirect;
