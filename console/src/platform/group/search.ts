// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type resource } from "@synnaxlabs/client";
import { Group } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { Tree } from "@/platform/tree";

/** The node that holds a group: its parent, or the root of its scope. */
const parentOf = (g: group.Group): resource.ID =>
  g.parent ?? { type: g.scope, key: "" };

const toGroupHit = (g: group.Group): Tree.Hit => ({
  ...Tree.toResource("group")(g),
  parent: parentOf(g),
});

/**
 * Returns a function that shows a record that groups can hold as a search hit. The hit
 * sits in the record's group, or at the root of its type when it has none.
 */
export const toHit =
  (type: resource.Type) =>
  <D extends record.KeyedNamed<record.Key> & { group?: group.Key }>(
    record: D,
  ): Tree.Hit => ({
    ...Tree.toResource(type)(record),
    parent: record.group == null ? { type, key: "" } : group.resourceID(record.group),
  });

/**
 * Matches groups by name. A root that names a type keeps the groups of that scope;
 * {@link usePlacement} drops the matches outside the root.
 */
export const useMatches: Tree.UseMatches = (query) => {
  const result = Group.useResultList(
    query == null
      ? null
      : {
          searchTerm: query.term,
          limit: query.limit,
          ...(query.root.key === "" ? { scope: query.root.type } : {}),
        },
  );
  return [Tree.toMatches(result, toGroupHit)];
};

/** Places each match under the groups between it and the root. */
export const usePlacement = Tree.createUsePlacement({
  type: "group",
  parentOf,
  useAncestors: Group.useResultAncestors,
});
