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
import { Sync } from "@/query/sync";

const matchLabelRelationship = (rel: ontology.Relationship, id: ontology.CrudeID) =>
  rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE && rel.from.equals(id);

export const useLabelsOf = (id: ontology.CrudeID): Query.UseReturn<label.Label[]> =>
  Query.use({
    name: "Labels",
    params: id,
    retrieve: async ({ client, params: id }) => await client.labels.retrieveFor(id),
    listeners: [
      {
        channel: label.SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(label.labelZ, async ({ changed, onChange }) =>
          onChange((prev) => [...prev.filter((l) => l.key !== changed.key), changed]),
        ),
      },
      {
        channel: label.DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ changed, onChange }) =>
          onChange((prev) => prev.filter((l) => l.key !== changed)),
        ),
      },
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
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
        onChange: Sync.stringHandler(async ({ changed, onChange, params: id }) => {
          const rel = ontology.parseRelationship(changed);
          if (!matchLabelRelationship(rel, id)) return;
          onChange((prev) => prev.filter((l) => l.key !== rel.to.key));
        }),
      },
    ],
  });

export const labelsOfFormSchema = z.object({ labels: z.array(label.keyZ) });

export const useLabelsOfForm = (id: ontology.CrudeID) =>
  Query.useForm<ontology.CrudeID, typeof labelsOfFormSchema>({
    name: "Labels",
    schema: labelsOfFormSchema,
    params: id,
    initialValues: { labels: [] },
    retrieve: async ({ client, params: id }) => {
      if (id == null) return null;
      const labels = await client.labels.retrieveFor(id);
      return { labels: labels.map((l) => l.key) };
    },
    update: async ({ client, values, params: id }) => {
      if (id == null) return { labels: [] };
      await client.labels.label(id, values.labels, { replace: true });
      return { labels: values.labels };
    },
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ client, changed, onChange, params: id }) => {
            const rel = ontology.parseRelationship(changed);
            if (!matchLabelRelationship(rel, id)) return;
            const { key } = rel.to;
            const l = await client.labels.retrieve(key);
            onChange((prev) => {
              if (prev == null) return { labels: [l.key] };
              return { labels: [...prev.labels.filter((l) => l !== key), l.key] };
            });
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ changed, onChange, params: id }) => {
          const rel = ontology.parseRelationship(changed);
          if (!matchLabelRelationship(rel, id)) return;
          onChange((prev) => {
            if (prev == null) return { labels: [] };
            return { labels: prev.labels.filter((l) => l !== rel.to.key) };
          });
        }),
      },
    ],
  });

export const useSetSynchronizer = (onSet: (label: label.Label) => void): void =>
  Sync.useListener({
    channel: label.SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(label.labelZ, async (args) => {
      onSet(args.changed);
    }),
  });

export const useDeleteSynchronizer = (onDelete: (key: label.Key) => void): void =>
  Sync.useListener({
    channel: label.DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(label.keyZ, async (args) => {
      onDelete(args.changed);
    }),
  });
