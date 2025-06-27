// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";
import { type primitive } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Query } from "@/query";
import { Sync } from "@/query/sync";

export const matchRelationship = (rel: ontology.Relationship, id: ontology.ID) =>
  rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
  ontology.idsEqual(rel.from, id);

interface UseLabelsOfQueryParams extends Record<string, primitive.Value> {
  id: ontology.ID;
}

export const useLabelsOf = (id: ontology.ID): Query.UseReturn<label.Label[]> =>
  Query.use<UseLabelsOfQueryParams, label.Label[]>({
    name: "Labels",
    params: { id },
    retrieve: async ({ client, params: { id } }) => await client.labels.retrieveFor(id),
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
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ client, changed, onChange, params: { id } }) => {
            if (!matchRelationship(changed, id)) return;
            const { key } = changed.to;
            const l = await client.labels.retrieve(key);
            onChange((prev) => [...prev.filter((l) => l.key !== key), l]);
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ changed, onChange, params: { id } }) => {
            if (!matchRelationship(changed, id)) return;
            onChange((prev) => prev.filter((l) => l.key !== changed.to.key));
          },
        ),
      },
    ],
  });

export const labelsOfFormSchema = z.object({ labels: z.array(label.keyZ) });

export const useLabelsOfForm = (id: ontology.ID) =>
  Query.useForm<UseLabelsOfQueryParams, typeof labelsOfFormSchema>({
    name: "Labels",
    schema: labelsOfFormSchema,
    params: { id },
    initialValues: { labels: [] },
    retrieve: async ({ client, params: { id } }) => {
      if (id == null) return null;
      const labels = await client.labels.retrieveFor(id);
      return { labels: labels.map((l) => l.key) };
    },
    update: async ({ client, values, params: { id } }) => {
      if (id == null) return;
      await client.labels.label(id, values.labels, {
        replace: true,
      });
    },
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
          async ({ client, changed, onChange, params: { id } }) => {
            if (!matchRelationship(changed, id)) return;
            const { key } = changed.to;
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
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
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
