// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Access, type Flux, Icon, List, Menu, Text, User } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";
import { User as PlatformUser } from "@/platform/user";

const useName: Tree.UseName = (id) =>
  User.useRetrieve({ key: id.key }).data?.username ?? "";

const useDelete = Tree.createUseDelete({
  type: "User",
  query: User.useDelete,
  convertKey: String,
  useName,
});

const useRename = ({
  selection: {
    ids: [firstID],
  },
}: Tree.ContextMenuProps): (() => void) => {
  const username = useName(firstID);
  const beforeUpdate = useCallback(
    async ({ data }: Flux.BeforeUpdateParams<User.ChangeUsernameParams>) => {
      const [username, renamed] = await Text.asyncEdit(
        List.itemNameID(ontology.idToString(firstID)),
      );
      if (!renamed) return false;
      return { ...data, username };
    },
    [firstID],
  );
  const { update } = User.useRename({ beforeUpdate });
  return useCallback(
    () => update({ key: firstID.key, username }),
    [update, firstID, username],
  );
};

const useAssignRole = (): ((props: Tree.ContextMenuProps, name: string) => void) => {
  const openAssignRole = PlatformUser.useAssignRoleModal();

  return useCallback(
    ({ selection: { ids } }: Tree.ContextMenuProps, name: string) => {
      openAssignRole({ userKey: ids[0].key, title: `Role.Assign.${name}` });
    },
    [openAssignRole],
  );
};

const retrieveProperties = async ({
  client,
  store,
  id,
}: Tree.RetrievePropertiesParams) =>
  await User.retrieveSingle({
    client,
    store: store as User.FluxSubStore,
    query: { key: id.key },
  });

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    selection: { ids },
  } = props;
  const name = useName(ids[0]);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const handleAssignRole = useAssignRole();
  const singleResource = ids.length === 1;
  const isNotCurrentUser = name !== client.params.username;
  const isRootUser = User.useRetrieve({ key: ids[0]?.key }).data?.rootUser === true;

  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && isNotCurrentUser && (
        <>
          <Menu.Item itemKey="rename" onClick={rename}>
            <Icon.Rename />
            Change username
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {hasUpdatePermission && singleResource && !isRootUser && isNotCurrentUser && (
        <>
          <Menu.Item itemKey="assignRole" onClick={() => handleAssignRole(props, name)}>
            <Icon.Role />
            Change role
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
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
  type: "user",
  icon: <Icon.User />,
  useName,
  ContextMenu: TreeContextMenu,
  hasChildren: false,
  haulItems: ({ id }, store) => {
    const user = (store as User.FluxSubStore).users.get(id.key);
    return user == null ? [] : [User.createHaulItem(user)];
  },
});

export const TREE_ITEMS = { user: TreeItem } satisfies Tree.Items;
