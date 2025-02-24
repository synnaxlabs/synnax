// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Tree } from "@synnaxlabs/pluto";

import { type Ontology } from "@/ontology";

export const getResourcesToGroup = (
  selection: Ontology.TreeContextMenuProps["selection"],
): ontology.ID[] => {
  const nodesOfMinDepth = Tree.getAllNodesOfMinDepth(selection.nodes);
  const nodesOfMinDepthKeys = nodesOfMinDepth.map(({ key }) => key);
  return selection.resources
    .filter(({ id }) => nodesOfMinDepthKeys.includes(id.toString()))
    .map(({ id }) => id);
};
