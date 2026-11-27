// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { opcua, status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, Device } from "@synnaxlabs/pluto";

import { useDelete, useRename, useRetrieve } from "@/feature/opcua/server/queries";
import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";

const useDeleteServer = Tree.createUseDelete({
  type: "OPC UA server",
  useUpdate: useDelete,
  convertKey: String,
});

const useRenameServer = Tree.createUseRename({
  useUpdate: useRename,
  resourceID: opcua.server.resourceID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
  } = props;
  const resourceIDs = ids.map((id) => opcua.server.resourceID(id.key));
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const handleDelete = useDeleteServer(props);
  const rename = useRenameServer(props);
  if (ids.length === 0) return null;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && ids.length === 1 && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, icon: _icon, ...rest }: Tree.ContentProps) => {
  const { itemKey } = rest;
  const { data: server } = useRetrieve({ key: resource.id.key });
  const highlight = Tree.useHighlight();
  return (
    <PTree.Item {...rest}>
      <Flex.Box x grow align="center">
        <Icon.Logo.OPCUA />
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          allowDoubleClick={false}
          value={resource.name}
          onChange
          overflow="fade"
          status={status.keepVariants(server?.status?.variant, "disabled")}
          highlight={highlight}
        />
        <Tree.MatchGroup id={resource.id} />
      </Flex.Box>
      <Device.StatusIndicator status={server?.status} />
    </PTree.Item>
  );
};

const TreeItem = Tree.createItem({
  icon: <Icon.Logo.OPCUA />,
  typeName: "OPC UA server",
  Content,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { opcua_server: TreeItem } satisfies Tree.Items;
