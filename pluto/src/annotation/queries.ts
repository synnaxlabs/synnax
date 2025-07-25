// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { annotation, type ontology, TimeRange } from "@synnaxlabs/client";
import z from "zod";

import { Flux } from "@/flux";

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
});

export const formSchema = z.object({
  key: annotation.keyZ.optional(),
  timeRange: z
    .object({
      start: z.number().optional(),
      end: z.number().optional(),
    })
    .optional(),
  message: z.string(),
});

const ZERO_FORM_VALUES = {
  key: undefined,
  timeRange: undefined,
  message: "",
};

interface UseFormParams {
  parent?: ontology.ID;
  key?: annotation.Key;
}

const annotationToFormValues = (annotation: annotation.Annotation) => ({
  key: annotation.key,
  timeRange: annotation.timeRange.numeric,
  message: annotation.message,
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
    if (params.parent == null) return;
    console.log("updating annotation", value);
    await client.annotations.create(
      {
        key: params.key,
        message: value.message,
        timeRange: TimeRange.z.parse(value.timeRange) ?? TimeRange.ZERO,
      },
      params.parent,
    );
  },
});
