// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, User } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { Tree } from "@/platform/tree";
import { User as PlatformUser } from "@/platform/user";

const useDelete = Tree.createUseDelete({
  type: "User",
  useUpdate: User.useDelete,
  convertKey: String,
});

const useRename = ({
  selection: {
    ids: [firstID],
  },
  state: { getResource },
}: Tree.ContextMenuProps): (() => void) => {
  const beforeUpdate = async ({
    data,
  }: Flux.BeforeUpdateParams<User.ChangeUsernameParams>) => {
    const [username, renamed] = await Text.asyncEdit(
      List.itemNameID(ontology.idToString(firstID)),
    );
    if (!renamed) return false;
    return { ...data, username };
  };
  const { update } = User.useRename({ beforeUpdate });
  return () => update({ key: firstID.key, username: getResource(firstID).name });
};

const useAssignRole = (): ((props: Tree.ContextMenuProps) => void) => {
  const openAssignRole = PlatformUser.useAssignRoleModal();

  return ({ selection: { ids }, state: { getResource } }: Tree.ContextMenuProps) => {
    const resource = getResource(ids[0]);
    openAssignRole({ userKey: ids[0].key, title: ["Role", "Assign", resource.name] });
  };
};

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    state,
    selection: { ids },
  } = props;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const handleAssignRole = useAssignRole();
  const openCredentials = Credential.useModal();
  const singleResource = ids.length === 1;
  const first = Tree.useResource(state, ids[0]);
  const isNotCurrentUser = first != null && first.name !== client.params.username;
  const isRootUser = first?.data?.root_user === true;
  const isSelf = ids[0]?.key === client.auth.subject?.key;

  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && isNotCurrentUser && (
        <>
          <Menu.Item itemKey="rename" onClick={rename}>
            <Icon.Rename />
            Change username
          </Menu.Item>
          {!isRootUser && (
            <Menu.Item itemKey="assignRole" onClick={() => handleAssignRole(props)}>
              <Icon.Role />
              Change role
            </Menu.Item>
          )}
        </>
      )}
      {singleResource && first != null && (hasUpdatePermission || isSelf) && (
        <Menu.Item
          itemKey="credentials"
          onClick={() =>
            openCredentials({ subject: ids[0], title: ["Credentials", first.name] })
          }
        >
          <Icon.Lock />
          Manage credentials
        </Menu.Item>
      )}
      <Menu.Divider />
      {singleResource && <Tree.CopyPropertiesContextMenuItem {...props} />}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "user",
  icon: <Icon.User />,
  ContextMenu: TreeContextMenu,
  hasChildren: false,
  haulItems: ({ data }) =>
    data == null ? [] : [User.createHaulItem(data as user.User)],
});

export const TREE_ITEMS = { user: TreeItem } satisfies Tree.Items;
