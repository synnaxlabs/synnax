// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { type Key, keyZ, type Label, labelZ } from "@/label/types.gen";
import { ontology } from "@/ontology";

export const SET_CHANNEL_NAME = "sy_label_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

export const STORE_KEY = "labels";

/** Reports whether the relationship labels the given ontology ID. */
export const matchLabeledBy = (rel: ontology.Relationship, id: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: id,
    type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

/** Returns the cached labels attached to the given ontology ID. */
export const cachedLabelsOf = (
  relationships: cache.Store<string, ontology.Relationship>,
  labels: cache.Store<Key, Label>,
  id: ontology.ID,
): Label[] =>
  labels.get(relationships.get((r) => matchLabeledBy(r, id)).map((r) => r.to.key));

/** Registers the label store on the given engine. */
export const bindStore = (
  engine: cache.Engine,
  retrieve: (keys: Key[]) => Promise<Label[]>,
): void => {
  const store = () => engine.store<Key, Label>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof labelZ> = {
    channel: SET_CHANNEL_NAME,
    schema: labelZ,
    onChange: ({ changed }) => store().set(changed.key, changed),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  engine.registerStore<Key, Label>(STORE_KEY, {
    listeners: [set, del],
    refetch: retrieve,
  });
};
