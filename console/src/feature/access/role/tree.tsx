// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type Synnax, user } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, User } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";

const useDelete = Tree.createUseDelete({
  type: "Role",
  useUpdate: Access.Role.useDelete,
  convertKey: String,
});

const useRename = Tree.createUseRename({
  useUpdate: Access.Role.useRename,
  resourceID: access.role.resourceID,
  convertKey: String,
});

const retrieve: Tree.Retrieve = async (c, key) =>
  await c.access.roles.retrieve({ key });

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
  } = props;
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const singleResource = ids.length === 1;
  const roles = Access.Role.useMultiple({ keys: ids.map(({ key }) => key) });
  const hasInternal = roles.some((r) => r.internal === true);
  return (
    <ContextMenu.Menu>
      {singleResource && hasUpdatePermission && !hasInternal && (
        <ContextMenu.RenameItem onClick={handleRename} />
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {singleResource && (
        <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
      )}
      <Menu.Divider />
      {hasDeletePermission && !hasInternal && (
        <ContextMenu.DeleteItem onClick={handleDelete} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

/** Lists the top level of the role tree: every role. */
export const listTreeRoot = (client: Synnax): Tree.Source[] => [
  Tree.createSource(client.access.roles, {}, Tree.toResource("role")),
];

const TreeItem = Tree.createItem({
  icon: Access.Role.icon,
  ContextMenu: TreeContextMenu,
  children: ({ client, id }) => [
    Tree.createSource(client.users, { roles: [id.key] }, Tree.toResource("user")),
  ],
  drop: async ({ client, destination, items }) =>
    await client.users.setRole(
      items.map(({ id }) => id.key),
      destination.id.key,
    ),
  canDrop: ({ items, moved }) =>
    moved.every(({ id }) => id.type === user.TYPE_RESOURCE_ID.type) &&
    User.filterHaulItems(items).every(({ data }) => !data.rootUser),
});

export const TREE_ITEMS = { role: TreeItem } satisfies Tree.Items;
