// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type group,
  NotFoundError,
  ontology,
  query,
  schematic,
} from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

/// isMissing reports whether a schematic symbol retrieve resolved to a missing
/// reference. Lets canvas and property-panel call sites branch on the existing
/// Flux Result without inventing a parallel state type.
export const isMissing = (res: Flux.Result<schematic.symbol.Symbol>): boolean =>
  res.variant === "error" && NotFoundError.matches(res.status.details.error);

const RESOURCE_NAME = "schematic symbol";
const PLURAL_RESOURCE_NAME = "schematic symbols";

export type RetrieveQuery = {
  key: string;
};

export const { useRetrieve, useRetrieveEffect } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.symbol.Symbol
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.schematics.symbols.retrieve(query),
  subscribe: ({ client, query }, handler) =>
    client.schematics.symbols.onChange(query, handler),
  getCached: ({ client, query }) => client.schematics.symbols.getCached(query),
});

export type ListQuery = {
  keys?: string[];
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
};

export const useList = Flux.createList<ListQuery, string, schematic.symbol.Symbol>({
  sort: (a, b) => a.name.localeCompare(b.name),
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.schematics.symbols.retrieve(query),
  retrieveByKey: async ({ client, key }) =>
    await client.schematics.symbols.retrieve({ key }),
  subscribe: ({ client, query }, handler) =>
    client.schematics.symbols.onChange(query, handler),
  subscribeByKey: ({ client, key }, handler) =>
    client.schematics.symbols.onChange({ key }, handler),
  getCached: ({ client, query }) => client.schematics.symbols.getCached(query),
});

export type FormQuery = {
  key?: string;
};

export const formSchema = schematic.symbol.symbolZ
  .partial({ key: true })
  .extend({ parent: ontology.idZ });

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
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
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    const symbol = await client.schematics.symbols.retrieve({ key });
    const parents = await client.ontology.parents.retrieve({
      ids: schematic.symbol.ontologyID(key),
    });
    reset({
      version: 1,
      name: symbol.name,
      data: symbol.data,
      key: symbol.key,
      parent: parents[0]?.id ?? ontology.ROOT_ID,
    });
  },
  update: async ({ client, value, reset }) => {
    const payload = value();
    const created = await client.schematics.symbols.create(payload);
    reset({ ...created, parent: payload.parent });
  },
  mountListeners: ({ client, query: { key }, reset, get }) => {
    if (key == null) return [];
    return client.schematics.symbols.onChange({ key }, (result) => {
      if (result == null || query.Deleted.matches(result)) return;
      reset({
        ...result,
        parent:
          get<ontology.ID>("parent", { optional: true })?.value ?? ontology.ROOT_ID,
      });
    });
  },
});

export interface RenameParams extends Pick<schematic.symbol.Symbol, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    await client.schematics.symbols.rename(data.key, data.name);
    return data;
  },
});

export type DeleteParams = schematic.symbol.Key | schematic.symbol.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.schematics.symbols.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const { useRetrieve: useRetrieveGroup } = Flux.createRetrieve<{}, group.Group>({
  name: RESOURCE_NAME,
  retrieve: async ({ client }) => await client.schematics.symbols.retrieveGroup(),
});
