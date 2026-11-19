// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type group } from "@synnaxlabs/client";
import { color as xcolor, deep, verbs } from "@synnaxlabs/x";
import { generateKeyBetween } from "fractional-indexing";
import z from "zod";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

const RESOURCE_NAME = "color";
const PLURAL_RESOURCE_NAME = "colors";

export type RetrieveQuery = color.RetrieveSingleParams;

export const { use: useRetrieve, useResult } = Flux.createRetrieve<
  RetrieveQuery,
  color.Named
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => await client.colors.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.colors.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.colors.watch(key).getCached(),
});

export type RetrieveMultipleQuery = {
  keys: color.Key[];
};

export const { use: useRetrieveMultiple, useResult: useResultMultiple } =
  Flux.createRetrieve<RetrieveMultipleQuery, color.Named[]>({
    name: PLURAL_RESOURCE_NAME,
    retrieve: async ({ client, query: { keys } }) => await client.colors.retrieve(keys),
    onChange: ({ client, query: { keys } }, handler) =>
      client.colors.watch(keys).onChange(handler),
    getCached: ({ client, query: { keys } }) => client.colors.watch(keys).getCached(),
  });

const { useEnsure } = Flux.createRetrieve<RetrieveQuery, color.Named>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => await client.colors.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.colors.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.colors.watch(key).getCached(),
});

export type ListQuery = color.RetrieveMultipleParams;

/** Every named color that the query matches, kept live. */
export const { useResult: useResultList } = Flux.createRetrieve<
  ListQuery,
  color.Named[]
>({
  name: PLURAL_RESOURCE_NAME,
  ...flux.listReads(({ client, query }) => client.colors.watch(query)),
});

/** The named colors that the Core ships, with their shipped values. */
export const { useResult: useResultBuiltins } = Flux.createRetrieve<
  Record<string, never>,
  color.Named[]
>({
  name: "built-in colors",
  retrieve: async ({ client }) => await client.colors.retrieveBuiltins(),
});

const nameZ = z.string().refine((name) => name.trim() !== "", "Name is required");

export const createFormSchema = color.namedZ
  .partial({ key: true })
  .extend({ name: nameZ });

export const createFormPath = deep.path<z.infer<typeof createFormSchema>>();

const ZERO_CREATE_FORM_VALUES: z.infer<typeof createFormSchema> = {
  name: "",
  light: xcolor.construct("#000000"),
};

/** A form that creates a named color when the caller saves it. */
export const useCreateForm = Flux.createForm<null, typeof createFormSchema>({
  name: RESOURCE_NAME,
  initialValues: ZERO_CREATE_FORM_VALUES,
  schema: createFormSchema,
  update: async ({ client, value, reset }) => {
    reset(await client.colors.create(value()));
  },
});

export const formSchema = color.namedZ
  .pick({ name: true, light: true, dark: true })
  .extend({ name: nameZ });

export const formPath = deep.path<z.infer<typeof formSchema>>();

/** A form over the name and values of a cached named color. */
export const useForm = Flux.createDocumentForm<
  color.Key,
  color.Named,
  typeof formSchema
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  domain: (client) => client.colors,
  useEnsure,
  toValues: ({ name, light, dark }) => ({ name, light, dark }),
});

export const { useUpdate: useCreate } = Flux.createUpdate<color.New, color.Named>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.colors.create(data),
});

export interface AppendParams extends Omit<color.New, "position"> {}

/**
 * Creates a named color after every color of its palette that has a position. It
 * writes no other color, so colors with no position stay after it.
 */
export const { useUpdate: useAppend } = Flux.createUpdate<AppendParams, color.Named>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    const where = data.group == null ? { grouped: false } : { groups: [data.group] };
    const last = (await client.colors.retrieve({ where }))
      .filter((c) => c.position != null)
      .sort(color.compare)
      .at(-1);
    return await client.colors.create({
      ...data,
      position: generateKeyBetween(last?.position ?? null, null),
    });
  },
});

export interface RenameParams extends Pick<color.Named, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    // A dispatch reaches the Core only for a cached named color.
    await client.colors.retrieve(key);
    await client.colors.dispatch(key, color.set({ path: ["name"], value: name }));
    return data;
  },
});

export interface MoveParams extends Pick<color.Named, "key"> {
  /** The fractional index key of the new place in the palette. */
  position: string;
}

/** Moves named colors to new places in their palettes, in one transaction. */
export const { useUpdate: useMove } = Flux.createUpdate<MoveParams[]>({
  name: PLURAL_RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.batch.send(
      data.map(({ key, position }) => client.colors.setPositionRequest(key, position)),
      { onOptimistic: async () => await onOptimisticComplete(data) },
    );
    return data;
  },
});

/** Writes the shipped name and values back onto a built-in named color. */
export const { useUpdate: useRestore } = Flux.createUpdate<color.Named>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, name, light, dark } = data;
    await client.colors.retrieve(key);
    await client.colors.dispatch(key, [
      color.set({ path: ["name"], value: name }),
      color.set({ path: ["light"], value: light }),
      dark == null
        ? color.remove({ path: [], key: "dark" })
        : color.set({ path: ["dark"], value: dark }),
    ]);
    return data;
  },
});

export type DeleteParams = color.Key | color.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.colors.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreatePaletteParams {
  key: group.Key;
  name: string;
}

/** Creates a palette with no colors. */
export const { useUpdate: useCreatePalette } = Flux.createUpdate<CreatePaletteParams>({
  name: "palette",
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    await client.groups.create({ ...data, scope: color.TYPE_RESOURCE_ID.type });
    return data;
  },
});

export interface DeletePaletteParams {
  palette: group.Key;
  /** The named colors in the palette. */
  colors: color.Key[];
}

/** Deletes a palette and its named colors in one transaction. */
export const { useUpdate: useDeletePalette } = Flux.createUpdate<DeletePaletteParams>({
  name: "palette",
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.batch.send(
      [
        client.colors.deleteRequest(data.colors),
        client.groups.deleteRequest(data.palette),
      ],
      { onOptimistic: async () => await onOptimisticComplete(data) },
    );
    return data;
  },
});
