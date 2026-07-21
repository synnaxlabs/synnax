// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { label } from "@/label";
import { ontology } from "@/ontology";
import {
  DELETE_CHANNEL_NAME,
  type Key,
  keyZ,
  ontologyID,
  SET_CHANNEL_NAME,
} from "@/status/payload";
import { type Status, statusZ } from "@/status/types.gen";

export const STORE_KEY = "statuses";

/** Registers the status table on the given cache. */
export const bindStore = (engine: cache.Cache): void => {
  const table = () => engine.table<Key, Status>(STORE_KEY);
  const cachedLabelsOf = (id: ontology.ID): label.Label[] => {
    const keys = engine
      .table<string, ontology.Relationship>(ontology.RELATIONSHIPS_STORE_KEY)
      .get((rel) =>
        ontology.matchRelationship(rel, {
          from: id,
          type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        }),
      )
      .map((rel) => rel.to.key);
    return engine.table<label.Key, label.Label>(label.STORE_KEY).get(keys);
  };
  const set: cache.ChannelListener<{}, ReturnType<typeof statusZ>> = {
    channel: SET_CHANNEL_NAME,
    schema: statusZ(),
    onChange: ({ changed }) =>
      table().set(changed.key, (p) => {
        const next = { ...p, ...changed };
        next.labels = cachedLabelsOf(ontologyID(changed.key));
        return next;
      }),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Status>(STORE_KEY, { listeners: [set, del] });
};
