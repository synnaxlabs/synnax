// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, type patch, type record, zod } from "@synnaxlabs/x";
import { type z } from "zod";

import { type actions } from "@/actions";
import { type batch } from "@/batch";
import { group } from "@/group";
import { type Key } from "@/project/types.gen";
import { query } from "@/query";

/** A visualization, which belongs to a project. */
export interface Owned {
  project?: Key;
}

/** The typed lines of a visualization Where that match by project. */
export const OWNER_LINES = {
  projects: ({ project }: Owned, keys: Key[]): boolean =>
    project != null && keys.includes(project),
};

/**
 * Payload of a dispatch entry that moves one visualization. Its actions set the project
 * of the visualization and remove its group.
 */
export interface MovePayload<K extends record.Key> extends actions.DispatchPayload<
  K,
  patch.Action
> {}

/**
 * Builds one batch entry per visualization that moves it out of its group and into the
 * project. The cache write of each entry moves its cached visualization.
 */
export interface MoveRequest<K extends record.Key> {
  (keys: K | K[], to: Key): batch.Request<MovePayload<K>>[];
}

/**
 * Returns a builder of move entries for the dispatch endpoint of a visualization type.
 * The cache write of an entry calls move with its key and project, and move returns the
 * rollback.
 */
export const createMoveRequest =
  <K extends record.Key>(
    endpoint: string,
    move: (keys: K[], project: Key) => destructor.Destructor,
  ): MoveRequest<K> =>
  (keys, to) =>
    array.toArray(keys).map((key) => ({
      endpoint,
      payload: {
        key,
        actions: [
          { type: "set", set: { path: ["project"], value: to, guards: [] } },
          { type: "remove", remove: { path: [], key: "group", guards: [] } },
        ],
      },
      apply: () => move([key], to),
    }));

/** A visualization that can sit in a group. */
export interface Movable extends Owned {
  group?: string;
}

/**
 * Moves each cached visualization under keys out of its group and into the project,
 * skipping keys the table does not hold. Returns a rollback that restores only the
 * project and group fields.
 */
export const moveCached = <K extends record.Key, V extends Movable>(
  table: query.Table<K, V>,
  keys: K[],
  project: Key,
): destructor.Destructor => {
  const prev = keys.map((key) => {
    const cached = table.get(key);
    return [key, { project: cached?.project, group: cached?.group }] as const;
  });
  keys.forEach((key) =>
    query.partialUpdate(table, key, { project, group: undefined } as Partial<V>),
  );
  return () =>
    prev.forEach(([key, fields]) =>
      query.partialUpdate(table, key, fields as Partial<V>),
    );
};

/**
 * Returns a builder of group move entries for the dispatch endpoint of a visualization
 * type. When the batch commits, each entry applies the actions that the Core adds, such
 * as a set of the project, to its cached visualization.
 */
export const createGroupMoveRequest = <K extends record.Key, V extends Movable, A>(
  endpoint: string,
  table: query.Table<K, V>,
  resZ: z.ZodType<{ actions: A[] }>,
  reduce: (doc: V, actions: A[]) => { next: V },
): group.MoveRequest<K> => {
  const move = group.createMoveRequest<K>(endpoint, (keys, to) =>
    group.moveCached(table, keys, to),
  );
  return (keys, to) =>
    move(keys, to).map((entry) => ({
      ...entry,
      applyResponse: (res) => {
        const { actions } = zod.parse(resZ, res, { label: "dispatch response" });
        table.set(entry.payload.key, (doc) =>
          doc == null ? undefined : reduce(doc, actions).next,
        );
      },
    }));
};
