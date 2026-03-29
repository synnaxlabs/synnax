// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, Menu, type Tree } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { canGroupSelection } from "@/group/canGroupSelection";

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
}: ContextMenuItemProps): ReactElement | null =>
  canGroupSelection(ids, shape, rootID) ? (
    <>
      <Menu.Item itemKey="group" {...rest}>
        <Icon.Group />
        Group selection
      </Menu.Item>
      {showBottomDivider && <Menu.Divider />}
    </>
  ) : null;
