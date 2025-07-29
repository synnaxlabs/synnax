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

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";

export interface UseListParams extends annotation.RetrieveRequest {
  parent: ontology.ID;
}

export const useList = Flux.createList<
  UseListParams,
  annotation.Key,
  annotation.Annotation
>({
  name: "Annotations",
  retrieve: async ({ params, client }) => {
    const children = await client.ontology.retrieveChildren(params.parent, {
      types: [annotation.ONTOLOGY_TYPE],
    });
    return await client.annotations.retrieve({ keys: children.map((c) => c.id.key) });
  },
  retrieveByKey: async ({ key, client }) => await client.annotations.retrieve({ key }),
  listeners: [
    {
      channel: annotation.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(annotation.annotationZ, ({ changed, onChange }) =>
        onChange(changed.key, (prev) => {
          if (prev == null) return null;
          return changed;
        }),
      ),
    },
    {
      channel: annotation.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(annotation.keyZ, ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { parent }, client }) => {
          if (
            changed.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
            ontology.idsEqual(changed.from, parent) &&
            changed.to.type === annotation.ONTOLOGY_TYPE
          ) {
            const annotation = await client.annotations.retrieve({
              key: changed.to.key,
            });
            onChange(annotation.key, annotation, { mode: "append" });
          }
        },
      ),
    },
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

const annotationToFormValues = (annotation: annotation.Annotation) => ({
  key: annotation.key,
  timeRange: annotation.timeRange.numeric,
  message: annotation.message,
  parent: undefined,
});

export const useForm = Flux.createForm<UseFormParams, typeof formSchema>({
  name: "Annotation",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ params, client }) => {
    if (params.key == null) return null;
    const annotation = await client.annotations.retrieve({ key: params.key });
    return annotationToFormValues(annotation);
  },
  update: async ({ params, client, value }) => {
    if (value.parent == null) return;
    let timeRange = TimeRange.z.parse(value.timeRange);
    if (timeRange.isZero) timeRange = TimeStamp.now().spanRange(0);
    await client.annotations.create(
      {
        key: params.key ?? value.key,
        message: value.message,
        timeRange,
      },
      value.parent,
    );
  },
});

export interface UseDeleteParams {
  key: annotation.Key;
}

export const useDelete = Flux.createUpdate<UseDeleteParams, void>({
  name: "Annotation",
  update: async ({ params, client }) => await client.annotations.delete(params.key),
}).useDirect;
