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

export const canGroupSelection = (
  selection: ontology.ID[],
  shape: Tree.Shape,
): boolean => {
  const strIDs = selection.map((id) => ontology.idToString(id));
  const filteredShape = Tree.filterShape(shape, (key) => strIDs.includes(key));
  return Tree.getAllNodesOfMinDepth(filteredShape).length > 1;
};
