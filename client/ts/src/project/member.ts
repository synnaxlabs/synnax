// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type patch, type record } from "@synnaxlabs/x";

import type { actions } from "@/actions";
import type { batch } from "@/batch";
import type { group } from "@/group";
import type { Key } from "@/project/types.gen";

/** A visualization, which belongs to a project. */
export interface Member {
  project?: Key;
}

/** The typed lines of a visualization Where that match by project. */
export const MEMBER_LINES = {
  projects: ({ project }: Member, keys: Key[]): boolean =>
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
 * project.
 */
export interface MoveRequest<K extends record.Key> {
  (keys: K | K[], to: Key): batch.Request<MovePayload<K>>[];
}

/** Returns a builder of move entries that builds each entry with build. */
export const createMoveRequest =
  <K extends record.Key>(build: group.EntryBuilder<K>): MoveRequest<K> =>
  (keys, to) =>
    array.toArray(keys).map((key) =>
      build(key, [
        { type: "set", set: { path: ["project"], value: to, guards: [] } },
        { type: "remove", remove: { path: [], key: "group", guards: [] } },
      ]),
    );

/** A visualization that can sit in a group. */
export interface Movable extends Member {
  group?: string;
}
