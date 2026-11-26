// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, ICD } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { useCreate } from "@/feature/icd/useCreate";
import { useImport } from "@/feature/icd/useImport";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import { Settings as PlatformSettings } from "@/platform/settings";

const COLUMNS: List.Column[] = [
  { name: "Name", width: "minmax(24rem, 1fr)" },
  { name: "Messages", width: "12rem" },
];

const useOpen = (): ((key: icd.Key) => void) => {
  const openTab = Panel.useOpenTab();
  return (key) => openTab({ variant: "resource", resource: icd.resourceID(key) });
};

const Item = (props: List.ItemProps<icd.Key>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<icd.Key, icd.ICD>(itemKey);
  const canRename = Access.useUpdateGranted(icd.resourceID(itemKey));
  const { update: rename } = ICD.useRename();
  const itemClass = PlatformSettings.useItemClass(itemKey);
  const open = useOpen();
  if (item == null) return null;
  return (
    <Select.Item<icd.Key>
      {...props}
      checkable
      align="center"
      aria-label={item.name}
      onActivate={() => open(itemKey)}
      className={itemClass}
    >
      <List.Cells>
        <List.Cell>
          <Text.MaybeEditable
            id={List.itemNameID(itemKey)}
            value={item.name}
            onChange={canRename && ((name) => rename({ key: itemKey, name }))}
            allowDoubleClick={false}
            overflow="fade"
          />
        </List.Cell>
        <List.Cell>
          <Text.Text level="small" color={9}>
            {item.messages.length}
          </Text.Text>
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const ICDsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<icd.Key, icd.ICD>();
  const icds = getItem?.(keys) ?? [];
  const ids = icd.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const open = useOpen();
  const handleImport = useImport();
  const handleLink = Core.useCopyLinkToClipboard();
  const confirm = Modals.useConfirmDelete({
    type: "ICD",
    plural: "ICDs",
    icon: "Library",
  });
  const { update: remove } = ICD.useDelete({
    beforeUpdate: async () => await confirm(icds),
  });
  const single = icds.length === 1 ? icds[0] : undefined;
  return (
    <ContextMenu.Menu>
      {single != null && (
        <Menu.Item itemKey="open" onClick={() => open(single.key)}>
          <Icon.Edit />
          Open
        </Menu.Item>
      )}
      {single != null && canUpdate && (
        <>
          <ContextMenu.RenameItem
            onClick={() => Text.edit(List.itemNameID(single.key))}
          />
          <Menu.Item itemKey="import" onClick={() => handleImport(single.key)}>
            <Icon.Import />
            Import
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      {single != null && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: single.name, resourceID: icd.resourceID(single.key) })
          }
        />
      )}
      <Menu.Divider />
      {icds.length > 0 && canDelete && (
        <ContextMenu.DeleteItem onClick={() => remove(icds.map((i) => i.key))} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = Component.renderProp(ICDsMenu);

/**
 * Every ICD, with its message count. A click opens the ICD in a tab, and the menu of
 * the selected ICDs renames, imports into, or deletes them.
 */
export const Settings = (): ReactElement => {
  const list = ICD.useList({ pageSize: 1000 });
  const canCreate = Access.useCreateGranted(icd.TYPE_RESOURCE_ID);
  const create = useCreate();
  return (
    <PlatformSettings.Table<icd.Key, icd.ICD>
      list={list}
      columns={COLUMNS}
      noun="ICDs"
      menu={menu}
      createLabel={canCreate ? "Create ICD" : undefined}
      onCreate={create}
    >
      {({ key, ...p }) => <Item key={key} {...p} />}
    </PlatformSettings.Table>
  );
};
