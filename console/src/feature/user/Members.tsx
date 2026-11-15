// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, user } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, User } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";
import { User as PlatformUser } from "@/platform/user";

const COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Role", width: "30rem", field: true },
];

const Member = (props: List.ItemProps<user.Key>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<user.Key, user.User>(itemKey);
  const itemClass = PlatformSettings.useItemClass(itemKey);
  const { data: selfKey } = User.useResultKey({});
  const canUpdate = Access.useUpdateGranted(user.resourceID(itemKey));
  const canSetRole = Access.useUpdateGranted(access.role.TYPE_RESOURCE_ID);
  const { update: rename } = User.useRename();
  const { update: setRole } = User.useSetRole();
  if (item == null) return null;
  const self = item.key === selfKey;
  // Until the current user is known, any line could be theirs, so every line locks.
  const locked = self || selfKey == null;
  const renamable = canUpdate && !locked;
  return (
    <Select.Item<user.Key>
      {...props}
      checkable
      align="center"
      aria-label={item.name}
      className={CSS.cls(CSS.BE("user-members", "item"), itemClass)}
    >
      <List.Cells>
        <List.Cell>
          <Flex.Box x align="center" gap="small">
            <Text.MaybeEditable
              id={List.itemNameID(item.key)}
              value={item.name}
              onChange={renamable && ((name) => rename({ key: item.key, name }))}
              allowDoubleClick={false}
              overflow="fade"
            />
            {self && <Badge.Badge>You</Badge.Badge>}
          </Flex.Box>
        </List.Cell>
        <List.Cell>
          <Access.Role.Select
            value={item.role}
            onChange={(role: access.role.Key) => setRole({ key: item.key, role })}
            variant="floating"
            triggerProps={{ variant: "shadow" }}
            disabled={!canUpdate || !canSetRole || locked || item.rootUser}
          />
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const MembersMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<user.Key, user.User>();
  const users = getItem?.(keys) ?? [];
  const { data: selfKey } = User.useResultKey({});
  const ids = user.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const openCredentials = Credential.useModal();
  const confirm = Modals.useConfirmDelete({ type: "User" });
  const { update: remove } = User.useDelete({
    beforeUpdate: async () => await confirm(users),
  });
  const single = users.length === 1 ? users[0] : undefined;
  const self = single?.key === selfKey;
  const locked = selfKey == null || users.some((u) => u.key === selfKey);
  // The Core refuses to delete the root user.
  const deletable = !locked && users.every((u) => !u.rootUser);
  return (
    <ContextMenu.Menu>
      {single != null && canUpdate && !locked && (
        <ContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(single.key))}
        />
      )}
      {single != null && (canUpdate || self) && (
        <Menu.Item
          itemKey="signIn"
          onClick={() =>
            openCredentials({
              subject: user.resourceID(single.key),
              title: ["Sign-in", single.name],
            })
          }
        >
          <Icon.Lock />
          Manage sign-in
        </Menu.Item>
      )}
      <Menu.Divider />
      {users.length > 0 && canDelete && deletable && (
        <ContextMenu.DeleteItem onClick={() => remove(users.map((u) => u.key))} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = Component.renderProp(MembersMenu);

/**
 * Every user, with their role. The menu of the selected users renames them, manages
 * their sign-in, or deletes them.
 */
export const Members = (): ReactElement => {
  const list = Flux.useList((client) => client.users, {
    name: "users",
    query: { order: { field: "name" } },
    pageSize: 100,
  });
  const openRegister = PlatformUser.useRegisterModal();
  const canCreate = Access.useCreateGranted(user.TYPE_RESOURCE_ID);
  return (
    <PlatformSettings.Table<user.Key, user.User>
      list={list}
      columns={COLUMNS}
      noun="users"
      menu={menu}
      createLabel={canCreate ? "Register user" : undefined}
      onCreate={() => openRegister()}
    >
      {({ key, ...p }) => <Member key={key} {...p} />}
    </PlatformSettings.Table>
  );
};
