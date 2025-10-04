// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { annotation, ontology, TimeRange, TimeStamp } from "@synnaxlabs/client";
import z from "zod";

import { type annotation as aetherAnnotation } from "@/annotation/aether";
import { Flux } from "@/flux";

const RESOURCE_NAME = "Annotation";
const PLURAL_RESOURCE_NAME = "Annotations";

export interface ListQuery extends annotation.RetrieveRequest {
  parent?: ontology.ID;
}

export const useList = Flux.createList<
  ListQuery,
  annotation.Key,
  annotation.Annotation,
  aetherAnnotation.SubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ query, client }) => {
    if (query.parent == null) return [];
    const children = await client.ontology.retrieveChildren(query.parent, {
      types: ["annotation"],
    });
    return await client.annotations.retrieve({ keys: children.map((c) => c.id.key) });
  },
  retrieveByKey: async ({ key, client }) => await client.annotations.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, query: { parent } }) => [
    store.annotations.onSet(async (changed) => {
      onChange(changed.key, (prev) => {
        if (prev == null) return null;
        return changed;
      });
    }),
    store.annotations.onDelete(async (key) => onDelete(key)),
    store.relationships.onSet(async (changed) => {
      if (
        parent != null &&
        changed.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
        ontology.idsEqual(changed.from, parent) &&
        changed.to.type === "annotation"
      )
        onChange(
          changed.to.key,
          (prev) => store.annotations.get(changed.to.key) ?? prev,
          { mode: "append" },
        );
    }),
  ],
});

export const formSchema = z.object({
  key: annotation.keyZ.optional(),
  timeRange: z
    .object({
      start: z.number().optional(),
      end: z.number().optional(),
    })
    .optional(),
  message: z.string().nonempty(),
  parent: ontology.idZ.optional(),
});

const ZERO_FORM_VALUES = {
  key: undefined,
  timeRange: undefined,
  message: "",
  parent: undefined,
};

interface FormQuery {
  key?: annotation.Key;
}

const toFormValues = (
  annotation: annotation.Annotation,
): z.output<typeof formSchema> => ({
  key: annotation.key,
  timeRange: annotation.timeRange.numeric,
  message: annotation.message,
  parent: undefined,
});

export const useForm = Flux.createForm<
  FormQuery,
  typeof formSchema,
  aetherAnnotation.SubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key }, ...form }) => {
    if (key == null) return undefined;
    const annotation = await client.annotations.retrieve({ key });
    form.reset(toFormValues(annotation));
  },
  update: async ({ client, store, ...form }) => {
    const { key, message, parent, ...rest } = form.value();
    if (parent == null) return;
    let timeRange = TimeRange.z.parse(rest.timeRange);
    if (timeRange.isZero) timeRange = TimeStamp.now().spanRange(0);
    const annotation = await client.annotations.create(
      { key, message, timeRange },
      parent,
    );
    store.annotations.set(annotation.key, annotation);
  },
});

export const { useUpdate: useDelete } = Flux.createUpdate<
  annotation.Key | annotation.Key[],
  aetherAnnotation.SubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ data, client }) => {
    await client.annotations.delete(data);
    return data;
  },
});
