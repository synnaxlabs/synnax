// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { toResource } from "@/platform/tree/children";
import { type Hit, type Placement, type UsePlacement } from "@/platform/tree/search";
import { type Resource } from "@/platform/tree/types";

export interface CreateUsePlacementParams<F extends record.KeyedNamed<string>> {
  /** The type of the folders that hold the matches. */
  type: resource.Type;
  /** Returns the node that holds the folder, or null when no node holds it. */
  parentOf: (folder: F) => resource.ID | null;
  /** Retrieves the folders with the keys and every folder above them. */
  useAncestors: Flux.UseResult<{ keys: string[] }, F[]>;
}

/** Returns a hook that places each match under the folders between it and the root. */
export const createUsePlacement = <F extends record.KeyedNamed<string>>({
  type,
  parentOf,
  useAncestors,
}: CreateUsePlacementParams<F>): UsePlacement => {
  const toFolder = toResource(type);
  const place = (hits: Hit[], folders: F[], root: resource.ID): Placement => {
    const byKey = new Map(folders.map((f) => [f.key, f]));
    const pathOf = (hit: Hit): F[] | null => {
      const path: F[] = [];
      let parent: resource.ID | null = hit.parent;
      while (parent != null && !resource.idsEqual(parent, root)) {
        if (parent.type !== type) return null;
        const folder = byKey.get(parent.key);
        if (folder == null) return null;
        path.unshift(folder);
        parent = parentOf(folder);
      }
      return parent == null ? null : path;
    };
    const paths: Record<string, resource.ID[][]> = {};
    const used = new Map<string, Resource>();
    hits.forEach((hit) => {
      const path = pathOf(hit);
      if (path == null) return;
      const shown = path.map(toFolder);
      shown.forEach((f) => used.set(f.key, f));
      paths[hit.key] = [shown.map(({ id }) => id)];
    });
    return { resources: [...used.values()], paths };
  };
  const usePlacement: UsePlacement = (query) => {
    const above = new Set<string>();
    query?.matches.forEach(({ parent }) => {
      if (parent.type === type && !resource.idsEqual(parent, query.root))
        above.add(parent.key);
    });
    const ancestors = useAncestors(above.size === 0 ? null : { keys: [...above] });
    if (query == null) return { variant: "loading" };
    const { matches, root } = query;
    if (above.size === 0)
      return { variant: "success", placement: place(matches, [], root) };
    if (ancestors.variant === "error")
      return { variant: "error", description: ancestors.status.description };
    if (ancestors.variant !== "success") return { variant: "loading" };
    return { variant: "success", placement: place(matches, ancestors.data, root) };
  };
  return usePlacement;
};
