// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { type batch } from "@/batch";
import { type Key, keyZ } from "@/project/types.gen";
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

/** Schema of a set-project request for visualizations keyed by the given schema. */
export const moveReqZ = <K extends z.ZodType<record.Key>>(key: K) =>
  z.object({ keys: key.array(), project: keyZ });

/** Payload of a set-project request. */
export interface MovePayload<K extends record.Key> {
  keys: K[];
  project: Key;
}

/**
 * Builds a request that moves the visualizations out of their group and into the
 * project. Its cache write moves the cached visualizations.
 */
export interface MoveRequest<K extends record.Key> {
  (keys: K | K[], to: Key): batch.Request<MovePayload<K>>;
}

/**
 * Returns a builder of set-project requests to the endpoint. A request's cache write
 * calls move with its keys and project, and move returns the rollback.
 */
export const createMoveRequest =
  <K extends record.Key>(
    endpoint: string,
    move: (keys: K[], project: Key) => destructor.Destructor,
  ): MoveRequest<K> =>
  (keys, to) => {
    const payload = { keys: array.toArray(keys), project: to };
    return { endpoint, payload, apply: () => move(payload.keys, payload.project) };
  };

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
