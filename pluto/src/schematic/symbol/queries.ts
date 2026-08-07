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
import { useMemo } from "react";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "schematic symbol";
const PLURAL_RESOURCE_NAME = "schematic symbols";

export type RetrieveQuery = {
  key: string;
};

export const { use, useResult } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.symbol.Symbol
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.schematics.symbols.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.schematics.symbols.onChange(query, handler),
  getCached: ({ client, query }) => client.schematics.symbols.getCached(query),
});

export interface Resolved {
  symbol?: schematic.symbol.Symbol;
  // missing reports whether the key points at a symbol that no longer exists
  // (or never did), as opposed to one still loading.
  missing: boolean;
}

/**
 * Resolves a symbol specification by key, kept live across edits and deletes.
 * Loading and missing are distinct: `symbol` is unset for both, `missing` is
 * true only once the reference is known to be dangling.
 */
export const useResolved = (key: string | null): Resolved => {
  const res = useResult(key == null ? null : { key });
  return useMemo(() => {
    if (res.variant !== "error") return { symbol: res.data, missing: false };
    const { error } = res.status.details;
    return {
      missing: NotFoundError.matches(error.cause) || Flux.DeletedError.matches(error),
    };
  }, [res]);
};

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
    await client.schematics.symbols.retrieve(key),
  onChange: ({ client, query }, handler) =>
    client.schematics.symbols.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) =>
    client.schematics.symbols.onChange(key, handler),
  getCached: ({ client, query }) => client.schematics.symbols.getCached(query),
});

export type FormQuery = {
  key: string;
};

export const formSchema = schematic.symbol.symbolZ
  .partial({ key: true })
  .extend({ parent: ontology.idZ });

const toFormValues = (
  symbol: schematic.symbol.Symbol,
  parents: ontology.Resource[],
): z.infer<typeof formSchema> => ({
  name: symbol.name,
  data: symbol.data,
  key: symbol.key,
  parent: parents[0]?.id ?? ontology.ROOT_ID,
});

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  initialValues: {
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
  retrieve: async ({ client, query: { key } }) => {
    // The two reads share only the key, so serializing them would double the
    // latency the editor suspends for.
    const [symbol, parents] = await Promise.all([
      client.schematics.symbols.retrieve(key),
      client.ontology.parents.retrieve({ ids: schematic.symbol.ontologyID(key) }),
    ]);
    return toFormValues(symbol, parents);
  },
  getCached: ({ client, query: { key } }) => {
    const symbol = client.schematics.symbols.getCached(key);
    if (!query.isLive(symbol)) return undefined;
    const parents = client.ontology.parents.getCached({
      ids: schematic.symbol.ontologyID(key),
    });
    if (!query.isLive(parents)) return undefined;
    return toFormValues(symbol, parents);
  },
  update: async ({ client, value, reset }) => {
    const payload = value();
    const created = await client.schematics.symbols.create(payload);
    reset({ ...created, parent: payload.parent });
  },
  mountListeners: ({ client, query: { key }, reset, get }) =>
    client.schematics.symbols.onChange(key, (result) => {
      if (!query.isLive(result)) return;
      reset({
        ...result,
        parent:
          get<ontology.ID>("parent", { optional: true })?.value ?? ontology.ROOT_ID,
      });
    }),
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

export const { useResult: useResultGroup } = Flux.createRetrieve<{}, group.Group>({
  name: RESOURCE_NAME,
  retrieve: async ({ client }) => await client.schematics.symbols.retrieveGroup(),
});
