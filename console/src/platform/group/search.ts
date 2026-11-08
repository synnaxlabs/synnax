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

import { Tree } from "@/platform/tree";

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
  return [Tree.toMatches(result, Tree.toResource("group"))];
};

const groupOf = ({ id, data }: Tree.Resource): group.Key | undefined => {
  if (id.type !== "group") return group.keyZ.optional().parse(data?.group);
  const parent = resource.idZ.optional().parse(data?.parent);
  return parent?.type === "group" ? parent.key : undefined;
};

const atTop = (r: Tree.Resource, root: resource.ID): boolean => {
  if (r.id.type !== "group") return root.key === "";
  const parent = resource.idZ.optional().parse(r.data?.parent);
  if (parent != null) return resource.idsEqual(parent, root);
  return root.key === "" && r.data?.scope === root.type;
};

// Places each resource under its groups, reading the group of a member and the parent
// of a group. The groups hold every group above the resources.
const place = (
  resources: Tree.Resource[],
  groups: group.Group[],
  root: resource.ID,
): Tree.Placement => {
  const byKey = new Map(groups.map((g) => [g.key, Tree.toResource("group")(g)]));
  const pathOf = (r: Tree.Resource): Tree.Resource[] | null => {
    const path: Tree.Resource[] = [];
    let top = r;
    for (let key = groupOf(r); key != null; key = groupOf(top)) {
      if (root.type === "group" && key === root.key) return path;
      const g = byKey.get(key);
      if (g == null) return null;
      path.unshift(g);
      top = g;
    }
    return atTop(top, root) ? path : null;
  };
  const paths: Record<string, resource.ID[][]> = {};
  const used = new Map<string, Tree.Resource>();
  resources.forEach((r) => {
    const path = pathOf(r);
    if (path == null) return;
    path.forEach((g) => used.set(g.key, g));
    paths[r.key] = [path.map(({ id }) => id)];
  });
  return { resources: [...used.values()], paths };
};

/** Places each match under the groups between it and the root. */
export const usePlacement: Tree.UsePlacement = (query) => {
  const above = [...new Set(query?.matches.map(groupOf).filter((key) => key != null))];
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
