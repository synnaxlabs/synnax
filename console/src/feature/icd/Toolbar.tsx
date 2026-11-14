// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, ICD } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { useCreate } from "@/feature/icd/useCreate";
import { useImport } from "@/feature/icd/useImport";
import { ContextMenu as Base } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Empty } from "@/platform/empty";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { type Nav } from "@/platform/nav";
import { Panel } from "@/platform/panel";
import { Toolbar } from "@/platform/toolbar";

const nameID = (key: icd.Key): string => `icd-name-${key}`;

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<icd.Key, icd.ICD>;
}

const ContextMenu = ({ keys, getItem }: ContextMenuProps): ReactElement => {
  const ids = icd.resourceID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const openResource = Panel.useOpenResource();
  const handleImport = useImport();
  const handleLink = Core.useCopyLinkToClipboard();
  const confirm = Modals.useConfirmDelete({
    type: "ICD",
    plural: "ICDs",
    icon: "Library",
  });
  const { update: del } = ICD.useDelete({
    beforeUpdate: async ({ data }: Flux.BeforeUpdateParams<ICD.DeleteParams>) => {
      const icds = getItem(array.toArray(data));
      return (await confirm(icds)) && data;
    },
  });
  const [first] = keys;
  const single = keys.length === 1;
  const firstItem = single ? getItem(first) : undefined;
  return (
    <Base.Menu>
      {firstItem != null && (
        <Menu.Item
          itemKey="open"
          onClick={() =>
            openResource({
              id: icd.resourceID(first),
              key: first,
              name: firstItem.name,
            })
          }
        >
          <Icon.Edit />
          Open
        </Menu.Item>
      )}
      {hasUpdatePermission && single && (
        <>
          <Base.RenameItem onClick={() => Text.edit(nameID(first))} />
          <Menu.Item itemKey="import" onClick={() => handleImport(first)}>
            <Icon.Import />
            Import
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      {firstItem != null && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: firstItem.name, resourceID: icd.resourceID(first) })
          }
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && keys.length > 0 && (
        <Base.DeleteItem onClick={() => del(keys)} />
      )}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};

interface ItemProps extends List.ItemProps<icd.Key> {
  onOpen: (key: icd.Key) => void;
}

const Item = ({ onOpen, ...rest }: ItemProps): ReactElement | null => {
  const { itemKey } = rest;
  const item = List.useItem<icd.Key, icd.ICD>(itemKey);
  const canRename = Access.useUpdateGranted(icd.resourceID(itemKey));
  const { update: rename } = ICD.useRename();
  if (item == null) return null;
  return (
    <Select.Item {...rest} onDoubleClick={() => onOpen(itemKey)}>
      <Icon.Library />
      <Text.MaybeEditable
        id={nameID(itemKey)}
        value={item.name}
        onChange={canRename ? (name) => rename({ key: itemKey, name }) : undefined}
        allowDoubleClick={false}
        overflow="fade"
      />
    </Select.Item>
  );
};

const Content = (): ReactElement => {
  const [selected, setSelected] = useState<icd.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();
  const create = useCreate();
  const canCreate = Access.useCreateGranted(icd.TYPE_RESOURCE_ID);
  const { data, getItem, subscribe, more, answered } = ICD.useList({ pageSize: 1000 });
  const handleOpen = (key: icd.Key) =>
    openTab({ variant: "resource", resource: icd.resourceID(key) });
  const menu: NonNullable<Menu.ContextMenuProps["menu"]> = (props) => (
    <ContextMenu {...props} getItem={getItem} />
  );
  return (
    <Menu.ContextMenu menu={menu} {...menuProps}>
      <Toolbar.Content className={menuProps.className}>
        <Toolbar.Header>
          <Toolbar.Title>
            <Icon.Library />
            ICDs
          </Toolbar.Title>
          {canCreate && (
            <Toolbar.Actions>
              <Toolbar.CreateAction tooltip="Create ICD" onClick={create} />
            </Toolbar.Actions>
          )}
        </Toolbar.Header>
        <Toolbar.Body>
          <Select.Frame
            multiple
            data={data}
            getItem={getItem}
            subscribe={subscribe}
            value={selected}
            onChange={setSelected}
            onFetchMore={more}
            replaceOnSingle
          >
            <List.Scroll full="y" onContextMenu={menuProps.open}>
              <List.Items<icd.Key, icd.ICD>
                emptyContent={
                  answered && (
                    <Empty.Action
                      message="No ICDs"
                      action={canCreate ? "Create ICD" : undefined}
                      onClick={create}
                    />
                  )
                }
              >
                {({ key, ...p }) => <Item key={key} {...p} onOpen={handleOpen} />}
              </List.Items>
            </List.Scroll>
          </Select.Frame>
        </Toolbar.Body>
      </Toolbar.Content>
    </Menu.ContextMenu>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "icd",
  icon: <Icon.Library />,
  content: <Content />,
  trigger: ["L"],
  tooltip: "ICDs",
  sizeBounds: { lower: 225, upper: 400 },
  initialSize: 300,
  useVisible: () => Access.useRetrieveGranted(icd.TYPE_RESOURCE_ID),
};
