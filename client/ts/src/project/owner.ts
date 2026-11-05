// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, primitive, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { type batch } from "@/batch";
import { type Key, keyZ } from "@/project/types.gen";
import { query } from "@/query";
import { ranger } from "@/ranger";

/** Retrieve request fields that filter visualizations by the project or range they
 * belong to. */
export const ownerFilterZ = z.object({
  /** Keeps visualizations that belong to any of these projects. */
  projects: keyZ.array().optional(),
  /** Keeps visualizations that belong to any of these ranges. */
  ranges: ranger.keyZ.array().optional(),
});
export interface OwnerFilter extends z.infer<typeof ownerFilterZ> {}

/** A visualization, which belongs to exactly one project or range. */
export interface Owned {
  project?: Key;
  range?: ranger.Key;
}

/** Returns a predicate that tells whether a visualization passes the filter. */
export const createOwnerMatcher = (
  filter: OwnerFilter,
): ((owned: Owned) => boolean) => {
  const projects = primitive.isNonZero(filter.projects)
    ? new Set(filter.projects)
    : null;
  const ranges = primitive.isNonZero(filter.ranges) ? new Set(filter.ranges) : null;
  return ({ project, range }) =>
    (projects == null || (project != null && projects.has(project))) &&
    (ranges == null || (range != null && ranges.has(range)));
};

/** Schema of a set-project request for visualizations keyed by the given schema. */
export const setProjectReqZ = <K extends z.ZodType<record.Key>>(key: K) =>
  z.object({ keys: key.array(), project: keyZ });

/** Payload of a set-project request. */
export interface SetProjectPayload<K extends record.Key> {
  keys: K[];
  project: Key;
}

/** Builds a set-project request that moves the visualizations into the project. */
export const setProjectRequest = <K extends record.Key>(
  endpoint: string,
  keys: K | K[],
  project: Key,
): batch.Request<SetProjectPayload<K>> => ({
  endpoint,
  payload: { keys: array.toArray(keys), project },
});

/**
 * Sets the project of each cached visualization under keys, skipping keys the table
 * does not hold. Returns a rollback that restores only the project field.
 */
export const setCachedProject = <K extends record.Key, V extends Owned>(
  table: query.Table<K, V>,
  keys: K[],
  project: Key,
): destructor.Destructor => {
  const prev = keys.map((key) => [key, table.get(key)?.project] as const);
  keys.forEach((key) => query.partialUpdate(table, key, { project } as Partial<V>));
  return () =>
    prev.forEach(([key, p]) =>
      query.partialUpdate(table, key, { project: p } as Partial<V>),
    );
};
