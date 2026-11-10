// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { isScope } from "@/platform/group/member";

export interface ContextMenuItemProps extends Omit<Menu.ItemProps, "itemKey"> {
  ids: resource.ID[];
  rootID: resource.ID;
  parentID: resource.ID | null;
  shape: Tree.Shape;
}

export const ContextMenuItem = ({
  ids,
  shape,
  rootID,
  parentID,
  ...rest
}: ContextMenuItemProps): ReactElement | null => {
  const hasCreatePermission = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  if (
    !hasCreatePermission ||
    parentID == null ||
    !canGroupSelection(ids, shape, rootID)
  )
    return null;
  return (
    <Menu.Item itemKey="group" {...rest}>
      <Icon.Group />
      Group selection
    </Menu.Item>
  );
};

const canGroupSelection = (
  selection: resource.ID[],
  shape: Tree.Shape,
  rootID: resource.ID,
): boolean => {
  if (!isScope(rootID.type)) return false;
  const strIDs = selection.map((id) => resource.idToString(id));
  const filteredShape = Tree.filterShape(shape, (key) => strIDs.includes(key));
  return Tree.getAllNodesOfMinDepth(filteredShape).length > 0;
};
