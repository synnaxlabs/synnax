// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Tree } from "@synnaxlabs/pluto";

export const getResourcesToGroup = (
  resourceIDs: ontology.ID[],
  shape: Tree.Shape,
): ontology.ID[] => {
  const strIDs = resourceIDs.map((id) => ontology.idToString(id));
  const nodesOfMinDepth = Tree.getAllNodesOfMinDepth(
    Tree.filterShape(shape, (key) => strIDs.includes(key)),
  );
  return resourceIDs.filter((id) => nodesOfMinDepth.includes(ontology.idToString(id)));
};
