// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type user } from "@synnaxlabs/client";
import { Access, type Flux, Icon, List, Menu, Text, User } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";
import { User as PlatformUser } from "@/platform/user";

const useDelete = Tree.createUseDelete({
  type: "User",
  query: User.useDelete,
  convertKey: String,
});

const useRename = ({
  selection: {
    ids: [firstID],
  },
  state: { getResource },
}: Tree.ContextMenuProps): (() => void) => {
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
    () => update({ key: firstID.key, username: getResource(firstID).name }),
    [update, firstID, getResource],
  );
};

const useAssignRole = (): ((props: Tree.ContextMenuProps) => void) => {
  const openAssignRole = PlatformUser.useAssignRoleModal();

  return useCallback(
    ({ selection: { ids }, state: { getResource } }: Tree.ContextMenuProps) => {
      const resource = getResource(ids[0]);
      openAssignRole({ userKey: ids[0].key, title: `Role.Assign.${resource.name}` });
    },
    [openAssignRole],
  );
};

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    state: { getResource },
    selection: { ids },
  } = props;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const handleAssignRole = useAssignRole();
  const singleResource = ids.length === 1;
  const isNotCurrentUser = getResource(ids[0]).name !== client.params.username;
  const isRootUser = getResource(ids[0]).data?.root_user === true;

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
          <Menu.Item itemKey="assignRole" onClick={() => handleAssignRole(props)}>
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
          <Tree.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export const TREE_ITEM = Tree.createItem({
  type: "user",
  icon: <Icon.User />,
  ContextMenu: TreeContextMenu,
  hasChildren: false,
  haulItems: ({ data }) =>
    data == null ? [] : [User.createHaulItem(data as user.User)],
});
