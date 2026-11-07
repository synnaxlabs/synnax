// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, resource } from "@synnaxlabs/client";
import { Store } from "@synnaxlabs/lyra/store";

import { type TreeState } from "@/platform/tree/types";

export interface ResourceSource extends Pick<
  TreeState,
  "getResource" | "subscribeResource"
> {}

/**
 * Returns the tree's resources for the given IDs and re-renders when any of them
 * changes. IDs the tree holds no resource for are left out.
 */
export const useResources = (
  { getResource, subscribeResource }: ResourceSource,
  ids: resource.ID[],
): ontology.Resource[] =>
  Store.useItems({
    keys: resource.idToString(ids),
    subscribe: subscribeResource,
    getItem: getResource,
  });

/**
 * Returns the tree's resource for the given ID, or undefined once the tree no longer
 * holds it, and re-renders when it changes.
 */
export const useResource = (
  source: ResourceSource,
  id: resource.ID,
): ontology.Resource | undefined => useResources(source, [id])[0];
