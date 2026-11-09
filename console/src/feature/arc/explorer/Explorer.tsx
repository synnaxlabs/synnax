// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Arc } from "@synnaxlabs/pluto";

import { Arc as PlatformArc } from "@/platform/arc";
import { ContextMenu } from "@/platform/context-menu";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";

const RootContextMenu = ({ root, state }: Tree.DefaultContextMenuProps) => {
  const createArc = PlatformArc.useCreate();
  const createDirectory = PlatformArc.useCreateDirectory({ parent: root, state, root });
  const canCreateArc = Access.useCreateGranted(arc.TYPE_RESOURCE_ID);
  const canCreateDirectory = Access.useCreateGranted(arc.directory.TYPE_RESOURCE_ID);
  return (
    <ContextMenu.Menu>
      {canCreateArc && (
        <Menu.Item itemKey="createArc" onClick={createArc}>
          <Icon.Add />
          Create arc
        </Menu.Item>
      )}
      {canCreateDirectory && (
        <Menu.Item itemKey="newDirectory" onClick={createDirectory}>
          <Icon.Group />
          New directory
        </Menu.Item>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export const Explorer: Panel.Content = () => {
  const rootKey = Arc.Directory.useRootKey({});
  return (
    <Tree.Tree
      root={arc.directory.resourceID(rootKey)}
      emptyContent={<Empty.Action message="No Arcs" />}
      rootContextMenu={RootContextMenu}
    />
  );
};
