// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";
import { z } from "zod";

import { Flux } from "@/flux";

export const matchRelationship = (rel: ontology.Relationship, id: ontology.ID) =>
  ontology.matchRelationship(rel, {
    from: id,
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

interface UseLabelsOfQueryParams {
  id: ontology.ID;
}

export const retrieveLabelsOf = Flux.createRetrieve<
  UseLabelsOfQueryParams,
  label.Label[]
>({
  name: "Labels",
  retrieve: async ({ client, params: { id } }) =>
    await client.labels.retrieve({ for: id }),
  listeners: [
    {
      channel: label.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(label.labelZ, async ({ changed, onChange }) =>
        onChange((prev) => [...prev.filter((l) => l.key !== changed.key), changed]),
      ),
    },
    {
      channel: label.DELETE_CHANNEL_NAME,
      onChange: Flux.stringHandler(async ({ changed, onChange }) =>
        onChange((prev) => prev.filter((l) => l.key !== changed)),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ client, changed, onChange, params: { id } }) => {
          if (!matchRelationship(changed, id)) return;
          const { key } = changed.to;
          const l = await client.labels.retrieve({ key });
          onChange((prev) => [...prev.filter((l) => l.key !== key), l]);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { id } }) => {
          if (!matchRelationship(changed, id)) return;
          onChange((prev) => prev.filter((l) => l.key !== changed.to.key));
        },
      ),
    },
  ],
});

export const labelsOfFormSchema = z.object({ labels: z.array(label.keyZ) });

export const useLabelsOfForm = Flux.createForm<
  UseLabelsOfQueryParams,
  typeof labelsOfFormSchema
>({
  name: "Labels",
  schema: labelsOfFormSchema,
  initialValues: { labels: [] },
  retrieve: async ({ client, params: { id } }) => {
    if (id == null) return null;
    const labels = await client.labels.retrieve({ for: id });
    return { labels: labels.map((l) => l.key) };
  },
  update: async ({ client, value, params: { id } }) => {
    await client.labels.label(id, value.labels, { replace: true });
  },
  listeners: [
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ client, changed, onChange, params: { id } }) => {
          if (!matchRelationship(changed, id)) return;
          const { key } = changed.to;
          const l = await client.labels.retrieve({ key });
          onChange((prev) => {
            if (prev == null) return { labels: [l.key] };
            return { labels: [...prev.labels.filter((l) => l !== key), l.key] };
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { id } }) => {
          if (!matchRelationship(changed, id)) return;
          onChange((prev) => {
            if (prev == null) return { labels: [] };
            return { labels: prev.labels.filter((l) => l !== changed.to.key) };
          });
        },
      ),
    },
  ],
});

export const useSetSynchronizer = (onSet: (label: label.Label) => void): void =>
  Flux.useListener({
    channel: label.SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(label.labelZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: label.Key) => void): void =>
  Flux.useListener({
    channel: label.DELETE_CHANNEL_NAME,
    onChange: Flux.parsedHandler(label.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });

export interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, label.Key, label.Label>({
  name: "Labels",
  retrieve: async ({ client, params }) =>
    await client.labels.retrieve({
      ...params,
      search: params.term,
    }),
  retrieveByKey: async ({ client, key }) => await client.labels.retrieve({ key }),
  listeners: [
    {
      channel: label.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(label.labelZ, async ({ changed, onChange }) => {
        onChange(changed.key, changed, { mode: "prepend" });
      }),
    },
    {
      channel: label.DELETE_CHANNEL_NAME,
      onChange: Sync.parsedHandler(label.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
  ],
});

interface FormParams {
  key?: label.Key;
}

export const formSchema = label.labelZ.partial({ key: true });

export const useForm = Flux.createForm<FormParams, typeof formSchema>({
  name: "Label",
  initialValues: {
    name: "",
    color: "#000000",
  },
  schema: formSchema,
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return null;
    const label = await client.labels.retrieve({ key });
    return label;
  },
  update: async ({ client, value, onChange }) =>
    onChange(await client.labels.create(value)),
  listeners: [
    {
      channel: label.SET_CHANNEL_NAME,
      onChange: Sync.parsedHandler(
        label.labelZ,
        async ({ changed, onChange, params }) => {
          if (params.key == null || changed.key !== params.key) return;
          onChange(changed);
        },
      ),
    },
  ],
});

export interface DeleteParams {
  key: label.Key;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "Label",
  update: async ({ client, params: { key } }) => await client.labels.delete(key),
}).useDirect;
