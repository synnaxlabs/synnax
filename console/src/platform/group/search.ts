// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, resource } from "@synnaxlabs/client";
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

// Places each hit under the groups between its parent and the root. The groups hold
// every group above the hits.
const place = (
  hits: Tree.Hit[],
  groups: group.Group[],
  root: resource.ID,
): Tree.Placement => {
  const byKey = new Map(groups.map((g) => [g.key, g]));
  const pathOf = (hit: Tree.Hit): group.Group[] | null => {
    const path: group.Group[] = [];
    let { parent } = hit;
    while (!resource.idsEqual(parent, root)) {
      if (parent.type !== "group") return null;
      const g = byKey.get(parent.key);
      if (g == null) return null;
      path.unshift(g);
      parent = parentOf(g);
    }
    return path;
  };
  const paths: Record<string, resource.ID[][]> = {};
  const used = new Map<string, Tree.Resource>();
  hits.forEach((hit) => {
    const path = pathOf(hit);
    if (path == null) return;
    const shown = path.map(Tree.toResource("group"));
    shown.forEach((g) => used.set(g.key, g));
    paths[hit.key] = [shown.map(({ id }) => id)];
  });
  return { resources: [...used.values()], paths };
};

const groupKeyOf = ({ parent }: Tree.Hit): group.Key | undefined =>
  parent.type === "group" ? parent.key : undefined;

/** Places each match under the groups between it and the root. */
export const usePlacement: Tree.UsePlacement = (query) => {
  const above = [
    ...new Set(query?.matches.map(groupKeyOf).filter((key) => key != null)),
  ];
  const ancestors = Group.useResultAncestors(
    above.length === 0 ? null : { keys: above },
  );
  if (query == null) return { variant: "loading" };
  const { matches, root } = query;
  if (above.length === 0)
    return { variant: "success", placement: place(matches, [], root) };
  if (ancestors.variant === "error")
    return { variant: "error", description: ancestors.status.description };
  if (ancestors.variant !== "success") return { variant: "loading" };
  return { variant: "success", placement: place(matches, ancestors.data, root) };
};
