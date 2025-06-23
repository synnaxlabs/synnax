// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";
import { z } from "zod/v4";

import { Query } from "@/query";

const matchLabelRelationship = (rel: ontology.Relationship, id: ontology.CrudeID) =>
  rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE && rel.from.equals(id);

export const useLabelsOf = Query.create<ontology.CrudeID, label.Label[]>({
  name: "Labels",
  queryFn: async ({ client, params: id }) => await client.labels.retrieveFor(id),
  listeners: [
    {
      channel: label.SET_CHANNEL_NAME,
      onChange: Query.parsedHandler(label.labelZ, async ({ changed, onChange }) =>
        onChange((prev) => [...prev.filter((l) => l.key !== changed.key), changed]),
      ),
    },
    {
      channel: label.DELETE_CHANNEL_NAME,
      onChange: Query.stringHandler(async ({ changed, onChange }) =>
        onChange((prev) => prev.filter((l) => l.key !== changed)),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Query.stringHandler(
        async ({ client, changed, onChange, params: id }) => {
          const rel = ontology.parseRelationship(changed);
          if (!matchLabelRelationship(rel, id)) return;
          const { key } = rel.to;
          const l = await client.labels.retrieve(key);
          onChange((prev) => [...prev.filter((l) => l.key !== key), l]);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Query.stringHandler(async ({ changed, onChange, params: id }) => {
        const rel = ontology.parseRelationship(changed);
        if (!matchLabelRelationship(rel, id)) return;
        onChange((prev) => prev.filter((l) => l.key !== rel.to.key));
      }),
    },
  ],
});

export const labelsOfFormSchema = z.object({ labels: z.array(label.keyZ) });

export const useLabelsOfForm = Query.createForm<
  ontology.CrudeID,
  typeof labelsOfFormSchema
>({
  name: "Labels",
  schema: labelsOfFormSchema,
  queryFn: async ({ client, params: id }) => {
    if (id == null) return null;
    const labels = await client.labels.retrieveFor(id);
    return { labels: labels.map((l) => l.key) };
  },
  mutationFn: async ({ client, values, key }) => {
    if (key == null) return;
    await client.labels.label(key, values.labels, { replace: true });
  },
  listeners: [
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Query.stringHandler(
        async ({ client, changed, onChange, params: id }) => {
          const rel = ontology.parseRelationship(changed);
          if (!matchLabelRelationship(rel, id)) return;
          const { key } = rel.to;
          const l = await client.labels.retrieve(key);
          onChange((prev) => ({
            labels: [...prev.labels.filter((l) => l !== key), l.key],
          }));
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Query.stringHandler(async ({ changed, onChange, params: id }) => {
        const rel = ontology.parseRelationship(changed);
        if (!matchLabelRelationship(rel, id)) return;
        onChange((prev) => ({
          labels: prev.labels.filter((l) => l !== rel.to.key),
        }));
      }),
    },
  ],
});

export const useSetSynchronizer = (onSet: (label: label.Label) => void): void =>
  Query.useParsedListener(label.SET_CHANNEL_NAME, label.labelZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: label.Key) => void): void =>
  Query.useParsedListener(label.DELETE_CHANNEL_NAME, label.keyZ, onDelete);
