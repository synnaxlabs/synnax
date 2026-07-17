// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access } from "@synnaxlabs/client";
import { Access, Icon, Menu, User } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";

const useName: Tree.UseName = (id) =>
  Access.Role.useRetrieve({ key: id.key }).data?.name ?? "";

const useDelete = Tree.createUseDelete({
  type: "Role",
  query: Access.Role.useDelete,
  convertKey: String,
  useName,
});

const useRename = Tree.createUseRename({
  query: Access.Role.useRename,
  ontologyID: access.role.ontologyID,
  convertKey: String,
});

const retrieveProperties = async ({
  client,
  store,
  id,
}: Tree.RetrievePropertiesParams) =>
  await Access.Role.retrieveSingle({
    client,
    store: store as Access.Role.FluxSubStore,
    query: { key: id.key },
  });

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
  } = props;
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const singleResource = ids.length === 1;
  const roles = Access.Role.useRetrieveMultiple({
    keys: ids.map((id) => id.key),
  }).data;
  const hasInternal = roles?.some((r) => r.internal === true) ?? false;
  return (
    <ContextMenu.Menu>
      {singleResource && !hasInternal && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} />
          <Menu.Divider />
        </>
      )}
      {!hasInternal && (
        <>
          <ContextMenu.DeleteItem onClick={handleDelete} />
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Tree.CopyPropertiesContextMenuItem
            {...props}
            retrieveProperties={retrieveProperties}
          />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "role",
  icon: <Icon.Role />,
  useName,
  ContextMenu: TreeContextMenu,
  hasChildren: true,
  canDrop: ({ items }) => {
    const users = User.filterHaulItems(items);
    return users.length === items.length && users.every(({ data }) => !data.rootUser);
  },
});

export const TREE_ITEMS = { role: TreeItem } satisfies Tree.Items;
