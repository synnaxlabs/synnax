// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, resource } from "@synnaxlabs/client";
import { Arc } from "@synnaxlabs/pluto";

import { Tree } from "@/platform/tree";

// The node that holds a directory or an Arc. An Arc with no directory sits in the root.
const parentOf = (
  key: arc.directory.Key | undefined,
  root: resource.ID,
): resource.ID => (key == null ? root : arc.directory.resourceID(key));

const useMatches: Tree.UseMatches = (query) => {
  const params = query == null ? null : { searchTerm: query.term, limit: query.limit };
  const directories = Arc.Directory.useResultList(params);
  const arcs = Arc.useResultList(params);
  if (query == null) return [{ variant: "loading" }];
  const { root } = query;
  const directoryMatches = Tree.toMatches(directories, (d) => ({
    ...Tree.toResource("arc_directory")(d),
    parent: parentOf(d.parent, root),
  }));
  return [
    directoryMatches.variant === "success"
      ? {
          ...directoryMatches,
          resources: directoryMatches.resources.filter(({ id }) => id.key !== root.key),
        }
      : directoryMatches,
    Tree.toMatches(arcs, (a) => ({
      ...Tree.toResource("arc")(a),
      parent: parentOf(a.directory, root),
    })),
  ];
};

// Places each hit under the directories between its parent and the root. The
// directories hold every directory above the hits.
const place = (
  hits: Tree.Hit[],
  directories: arc.directory.Directory[],
  root: resource.ID,
): Tree.Placement => {
  const byKey = new Map(directories.map((d) => [d.key, d]));
  const pathOf = (hit: Tree.Hit): arc.directory.Directory[] | null => {
    const path: arc.directory.Directory[] = [];
    let { parent } = hit;
    while (!resource.idsEqual(parent, root)) {
      if (parent.type !== "arc_directory") return null;
      const d = byKey.get(parent.key);
      if (d?.parent == null) return null;
      path.unshift(d);
      parent = arc.directory.resourceID(d.parent);
    }
    return path;
  };
  const paths: Record<string, resource.ID[][]> = {};
  const used = new Map<string, Tree.Resource>();
  hits.forEach((hit) => {
    const path = pathOf(hit);
    if (path == null) return;
    const shown = path.map(Tree.toResource("arc_directory"));
    shown.forEach((d) => used.set(d.key, d));
    paths[hit.key] = [shown.map(({ id }) => id)];
  });
  return { resources: [...used.values()], paths };
};

const usePlacement: Tree.UsePlacement = (query) => {
  const above = [
    ...new Set(
      query?.matches
        .map(({ parent }) => parent)
        .filter(({ type, key }) => type === "arc_directory" && key !== query.root.key)
        .map(({ key }) => key),
    ),
  ];
  const ancestors = Arc.Directory.useResultAncestors(
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

/** Searches the Arcs and directories of the tree under the root directory. */
export const useSearch = Tree.createSearch({ noun: "Arcs", useMatches, usePlacement });
