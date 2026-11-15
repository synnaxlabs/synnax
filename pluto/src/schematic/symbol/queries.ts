// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  actions,
  type group,
  NotFoundError,
  query,
  schematic,
} from "@synnaxlabs/client";
import { deep, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "schematic symbol";
const GROUP_RESOURCE_NAME = "schematic symbol group";

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
    client.schematics.symbols.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.schematics.symbols.watch(query).getCached(),
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
  if (res.variant !== "error") return { symbol: res.data, missing: false };
  const { error } = res.status.details;
  return {
    missing: NotFoundError.matches(error.cause) || Flux.DeletedError.matches(error),
  };
};

/** Lists the symbol groups, which are top-level groups of the symbol scope. */
export const GROUP_QUERY = {
  scope: schematic.symbol.TYPE_RESOURCE_ID.type,
  nested: false,
} satisfies group.RetrieveMultipleParams;

export type ListQuery = schematic.symbol.RetrieveMultipleParams;

/** Lists symbols by name, unless the query sets an order. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, schematic.symbol.Symbol>,
  "name"
> = {}): Flux.UseListReturn<schematic.symbol.Key, schematic.symbol.Symbol> =>
  Flux.useList((client) => client.schematics.symbols, {
    ...rest,
    name: "symbols",
    query: query === null ? null : { order: { field: "name" }, ...query },
  });

export type FormQuery = {
  key: string;
};

export const formSchema = schematic.symbol.symbolZ.partial({ key: true });

export const formPath = deep.path<z.infer<typeof formSchema>>();

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
      strokeScaled: false,
      previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
    },
  },
  schema: formSchema,
  retrieve: async ({ client, query: { key } }) =>
    await client.schematics.symbols.retrieve(key),
  getCached: ({ client, query: { key } }) => {
    const symbol = client.schematics.symbols.watch(key).getCached();
    return query.isLive(symbol) ? symbol : undefined;
  },
  // A caller can set the key of a new symbol, so the form creates until it stores one.
  update: async ({ client, value, reset, stored }) => {
    const values = value();
    if (stored == null || values.key == null) {
      reset(await client.schematics.symbols.create(values));
      return;
    }
    const edits = actions.diffFields(formSchema, stored, values, ["name", "data"]);
    if (edits.length > 0) await client.schematics.symbols.dispatch(values.key, edits);
  },
  mountListeners: ({ client, query: { key }, reset }) =>
    client.schematics.symbols.watch(key).onChange((result) => {
      if (query.isLive(result)) reset(result);
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

export type DeleteGroupParams = group.Key;

export const { useUpdate: useDeleteGroup } = Flux.createUpdate<DeleteGroupParams>({
  name: GROUP_RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.schematics.symbols.deleteGroup(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
