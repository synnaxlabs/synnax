// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/user/Members.css";

import { access, user } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, User } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";
import { User as PlatformUser } from "@/platform/user";

interface SetRoleParams {
  key: user.Key;
  role: access.role.Key;
}

const { useUpdate: useSetRole } = Flux.createUpdate<SetRoleParams>({
  name: "user role",
  verbs: { present: "change", past: "changed", participle: "changing" },
  update: async ({ client, data }) => {
    await client.users.setRole(data.key, data.role);
    return data;
  },
});

const COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Role", width: "30rem", field: true },
  { width: "5rem" },
];

const nameID = (key: user.Key): string => `user-name-${key}`;

interface MemberProps extends List.ItemProps<user.Key> {
  onOpenMenu: Menu.ContextMenuOpen;
}

const Member = ({ onOpenMenu, ...props }: MemberProps): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<user.Key, user.User>(itemKey);
  const { data: currentKey } = User.useResultKey({});
  const canUpdate = Access.useUpdateGranted(user.resourceID(itemKey));
  const canSetRole = Access.useUpdateGranted(access.role.TYPE_RESOURCE_ID);
  const { update: rename } = User.useRename();
  const { update: setRole } = useSetRole();
  if (item == null) return null;
  const self = item.key === currentKey;
  const renamable = canUpdate && !self;
  return (
    <List.Item {...props} className={CSS.BE("user-members", "item")}>
      <List.Cells>
        <List.Cell>
          <Flex.Box x align="center" gap="small">
            <Text.MaybeEditable
              id={nameID(item.key)}
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
            disabled={!canUpdate || !canSetRole || self || item.rootUser}
          />
        </List.Cell>
        <List.Cell>
          <Button.Button
            variant="text"
            size="small"
            tooltip="More actions"
            onClick={onOpenMenu}
          >
            <Icon.KebabMenu />
          </Button.Button>
        </List.Cell>
      </List.Cells>
    </List.Item>
  );
};

interface MemberMenuProps {
  userKey: user.Key;
}

const MemberMenu = ({ userKey }: MemberMenuProps): ReactElement | null => {
  const item = List.useItem<user.Key, user.User>(userKey);
  const { data: currentKey } = User.useResultKey({});
  const canUpdate = Access.useUpdateGranted(user.resourceID(userKey));
  const canDelete = Access.useDeleteGranted(user.resourceID(userKey));
  const openCredentials = Credential.useModal();
  const confirm = Modals.useConfirmDelete({ type: "User" });
  const { update: remove } = User.useDelete({
    beforeUpdate: async () => item != null && (await confirm(item)),
  });
  if (item == null) return null;
  const self = item.key === currentKey;
  return (
    <ContextMenu.Menu>
      {canUpdate && !self && (
        <ContextMenu.RenameItem onClick={() => Text.edit(nameID(item.key))} />
      )}
      {(canUpdate || self) && (
        <Menu.Item
          itemKey="credentials"
          onClick={() =>
            openCredentials({
              subject: user.resourceID(item.key),
              title: ["Credentials", item.name],
            })
          }
        >
          <Icon.Lock />
          Manage credentials
        </Menu.Item>
      )}
      <Menu.Divider />
      {canDelete && !self && (
        <ContextMenu.DeleteItem onClick={() => remove(item.key)} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = ({ keys: [key] }: Menu.ContextMenuMenuProps): ReactElement =>
  key == null ? (
    <ContextMenu.Menu>
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  ) : (
    <MemberMenu userKey={key} />
  );

/**
 * Every user, with their role. A user's menu renames them, manages their credentials,
 * or deletes them.
 */
export const Members = (): ReactElement => {
  const { data, getItem, subscribe, more, search } = Flux.useList(
    (client) => client.users,
    { name: "users", query: { order: { field: "name" } }, pageSize: 100 },
  );
  const [term, setTerm] = useState("");
  const menuProps = Menu.useContextMenu();
  const openRegister = PlatformUser.useRegisterModal();
  const canCreate = Access.useCreateGranted(user.TYPE_RESOURCE_ID);
  const handleSearch = (next: string): void => {
    setTerm(next);
    search(next);
  };
  return (
    <List.Frame<user.Key, user.User>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      virtual={false}
    >
      <Menu.ContextMenu menu={menu} {...menuProps} />
      <Flex.Box y gap="medium">
        <Flex.Box x align="center" justify="between" gap="medium">
          <Input.Text
            value={term}
            onChange={handleSearch}
            placeholder="Search users"
            startContent={<Icon.Search />}
            className={CSS.BE("user-members", "search")}
          />
          {canCreate && (
            <Button.Button variant="outlined" onClick={() => openRegister()}>
              <Icon.Add />
              Create user
            </Button.Button>
          )}
        </Flex.Box>
        <PlatformSettings.Card flush>
          <List.Columns columns={COLUMNS}>
            <List.Headings />
            <List.Scroll onContextMenu={menuProps.open}>
              <List.Items<user.Key>>
                {({ key, ...p }) => (
                  <Member key={key} {...p} onOpenMenu={menuProps.open} />
                )}
              </List.Items>
            </List.Scroll>
          </List.Columns>
        </PlatformSettings.Card>
      </Flex.Box>
    </List.Frame>
  );
};
