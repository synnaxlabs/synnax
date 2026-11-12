// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions, query, view } from "@synnaxlabs/client";
import { deep, type optional, verbs } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "view";

export const { useUpdate: useCreate } = Flux.createUpdate<view.New>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.views.create(data),
});

export type DeleteParams = view.Key | view.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.views.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type FormValues =
  optional.Optional<view.RangeView, "key"> | optional.Optional<view.StatusView, "key">;

const nameZ = z.string().trim().min(1, "Name is required");

// TypeScript cannot name the recursive where types inside an inferred schema type, so
// the schema carries its value type by hand.
export const formSchema: z.ZodType<FormValues> = z.discriminatedUnion("type", [
  view.rangeViewZ.partial({ key: true }).extend({ name: nameZ }),
  view.statusViewZ.partial({ key: true }).extend({ name: nameZ }),
]);

export const formPath = deep.path<FormValues>();

const ZERO_VALUES: FormValues = {
  name: "",
  type: "range",
  sort: "newest",
  layout: "tree",
};
export type FormQuery = view.RetrieveSingleParams;

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_VALUES,
  retrieve: async ({ client, query: { key } }) => await client.views.retrieve(key),
  getCached: ({ client, query: { key } }) => {
    const cached = client.views.watch(key).getCached();
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, reset, stored }) => {
    const values = value();
    if (values.key == null) {
      reset(await client.views.create(values));
      return;
    }
    const edits = actions.diffFields(formSchema, stored, values, ["name", "where"]);
    if (edits.length > 0) await client.views.dispatch(values.key, edits);
  },
  mountListeners: ({ client, query: { key }, reset }) =>
    client.views.watch(key).onChange((result) => {
      if (query.isLive(result)) reset(result);
    }),
});

export interface RenameParams extends Pick<view.View, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.views.rename(key, name);
    return data;
  },
});

export interface SetFieldsParams {
  key: view.Key;
  actions: view.Action | view.Action[];
}

/** Sets fields of a view, such as its icon or its color. */
export const { useUpdate: useSetFields } = Flux.createUpdate<SetFieldsParams>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    await client.views.dispatch(data.key, data.actions);
    return data;
  },
});

export const useSetSynchronizer = (onSet: (view: view.View) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.views.onSet(onSet), [client, onSet]);
};

export const useDeleteSynchronizer = (onDelete: (key: view.Key) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.views.onDelete(onDelete), [client, onDelete]);
};
