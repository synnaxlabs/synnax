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

export interface UseListParams extends annotation.RetrieveRequest {
  parent: ontology.ID;
}

export const useList = Flux.createList<
  UseListParams,
  annotation.Key,
  annotation.Annotation,
  aetherAnnotation.SubStore
>({
  name: "Annotations",
  retrieve: async ({ params, client }) => {
    const children = await client.ontology.retrieveChildren(params.parent, {
      types: ["annotation"],
    });
    return await client.annotations.retrieve({ keys: children.map((c) => c.id.key) });
  },
  retrieveByKey: async ({ key, client }) => await client.annotations.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, params: { parent } }) => [
    store.annotations.onSet(async (changed) => {
      onChange(changed.key, (prev) => {
        if (prev == null) return null;
        return changed;
      });
    }),
    store.annotations.onDelete(async (key) => onDelete(key)),
    store.relationships.onSet(async (changed) => {
      if (
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

interface UseFormParams {
  key?: annotation.Key;
}

const annotationToFormValues = (
  annotation: annotation.Annotation,
): z.output<typeof formSchema> => ({
  key: annotation.key,
  timeRange: annotation.timeRange.numeric,
  message: annotation.message,
  parent: undefined,
});

export const useForm = Flux.createForm<
  UseFormParams,
  typeof formSchema,
  aetherAnnotation.SubStore
>({
  name: "Annotation",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return undefined;
    const annotation = await client.annotations.retrieve({ key });
    return annotationToFormValues(annotation);
  },
  update: async ({ params, client, value, store }) => {
    if (value.parent == null) return;
    let timeRange = TimeRange.z.parse(value.timeRange);
    if (timeRange.isZero) timeRange = TimeStamp.now().spanRange(0);
    const annotation = await client.annotations.create(
      {
        key: params.key ?? value.key,
        message: value.message,
        timeRange,
      },
      value.parent,
    );
    store.annotations.set(annotation.key, annotation);
  },
});

export interface UseDeleteParams {
  key: annotation.Key;
}

export const useDelete = Flux.createUpdate<
  UseDeleteParams,
  void,
  aetherAnnotation.SubStore
>({
  name: "Annotation",
  update: async ({ params, client }) => await client.annotations.delete(params.key),
}).useDirect;
