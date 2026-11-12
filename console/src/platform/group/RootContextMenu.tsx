// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { isScope } from "@/platform/group/member";
import { useCreateEmpty } from "@/platform/group/useCreateEmpty";
import { type Tree } from "@/platform/tree";

/** The menu for a right-click on the empty space of a tree. */
export const RootContextMenu = ({
  root,
  state,
}: Tree.RootContextMenuProps): ReactElement => {
  const createGroup = useCreateEmpty({ parent: root, state, root });
  const hasCreatePermission = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  return (
    <ContextMenu.Menu>
      {hasCreatePermission && isScope(root.type) && (
        <Menu.Item itemKey="newGroup" onClick={createGroup}>
          <Icon.Group />
          New group
        </Menu.Item>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};
