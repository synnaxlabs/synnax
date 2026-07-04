// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Access, Icon, Menu, Tree } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface ContextMenuItemProps extends Omit<Menu.ItemProps, "itemKey"> {
  ids: ontology.ID[];
  rootID: ontology.ID;
  shape: Tree.Shape;
  showBottomDivider?: boolean;
}

export const ContextMenuItem = ({
  ids,
  shape,
  showBottomDivider = false,
  rootID,
  ...rest
}: ContextMenuItemProps): ReactElement | null => {
  const hasCreatePermission = Access.useCreateGranted(group.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission || !canGroupSelection(ids, shape, rootID)) return null;
  return (
    <>
      <Menu.Item itemKey="group" {...rest}>
        <Icon.Group />
        Group selection
      </Menu.Item>
      {showBottomDivider && <Menu.Divider />}
    </>
  );
};

const canGroupSelection = (
  selection: ontology.ID[],
  shape: Tree.Shape,
  rootID: ontology.ID,
): boolean => {
  const strIDs = selection.map((id) => ontology.idToString(id));
  const filteredShape = Tree.filterShape(shape, (key) => strIDs.includes(key));
  const nodeKeysOfMinDepth = Tree.getAllNodesOfMinDepth(filteredShape);
  if (nodeKeysOfMinDepth.length < 1) return false;
  const isZeroDepth =
    Tree.getDepth(nodeKeysOfMinDepth[0], shape) === 0 &&
    ontology.idsEqual(rootID, ontology.ROOT_ID);
  return !isZeroDepth;
};
