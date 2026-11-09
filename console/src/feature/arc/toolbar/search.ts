// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type resource } from "@synnaxlabs/client";
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

const usePlacement = Tree.createUsePlacement({
  type: "arc_directory",
  parentOf: (d: arc.directory.Directory) =>
    d.parent == null ? null : arc.directory.resourceID(d.parent),
  useAncestors: Arc.Directory.useResultAncestors,
});

/** Searches the Arcs and directories of the tree under the root directory. */
export const useSearch = Tree.createSearch({ noun: "Arcs", useMatches, usePlacement });
