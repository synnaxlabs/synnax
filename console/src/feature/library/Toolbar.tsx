// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, Library } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { useCreate } from "@/feature/library/useCreate";
import { useImport } from "@/feature/library/useImport";
import { ContextMenu as Base } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Empty } from "@/platform/empty";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { type Nav } from "@/platform/nav";
import { Panel } from "@/platform/panel";
import { Toolbar } from "@/platform/toolbar";

const nameID = (key: library.Key): string => `library-name-${key}`;

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<library.Key, library.Library>;
}

const ContextMenu = ({ keys, getItem }: ContextMenuProps): ReactElement => {
  const ids = library.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const openResource = Panel.useOpenResource();
  const handleImport = useImport();
  const handleLink = Core.useCopyLinkToClipboard();
  const confirm = Modals.useConfirmDelete({ type: "Library" });
  const { update: del } = Library.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<Library.DeleteParams>) => {
        const libraries = getItem(array.toArray(data));
        return (await confirm(libraries)) && data;
      },
      [getItem, confirm],
    ),
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
              id: library.ontologyID(first),
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
            handleLink({ name: firstItem.name, ontologyID: library.ontologyID(first) })
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

interface ItemProps extends List.ItemProps<library.Key> {
  onOpen: (key: library.Key) => void;
}

const Item = ({ onOpen, ...rest }: ItemProps): ReactElement | null => {
  const { itemKey } = rest;
  const item = List.useItem<library.Key, library.Library>(itemKey);
  const canRename = Access.useUpdateGranted(library.ontologyID(itemKey));
  const { update: rename } = Library.useRename();
  if (item == null) return null;
  return (
    <Select.Item {...rest} onDoubleClick={() => onOpen(itemKey)}>
      <Icon.Library />
      <Text.MaybeEditable
        id={nameID(itemKey)}
        value={item.name}
        onChange={canRename ? (name) => rename({ key: itemKey, name }) : undefined}
        allowDoubleClick={false}
        overflow="ellipsis"
      />
    </Select.Item>
  );
};

const Content = (): ReactElement => {
  const [selected, setSelected] = useState<library.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();
  const create = useCreate();
  const canCreate = Access.useCreateGranted(library.TYPE_ONTOLOGY_ID);
  const { data, getItem, subscribe, retrieve, answered } = Library.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });
  const handleOpen = useCallback(
    (key: library.Key) =>
      openTab({ variant: "resource", resource: library.ontologyID(key) }),
    [openTab],
  );
  const menu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );
  return (
    <Menu.ContextMenu menu={menu} {...menuProps}>
      <Toolbar.Content className={menuProps.className}>
        <Toolbar.Header>
          <Toolbar.Title>
            <Icon.Library />
            Libraries
          </Toolbar.Title>
          {canCreate && (
            <Toolbar.Actions>
              <Toolbar.Action
                tooltip="Create library"
                onClick={create}
                variant="filled"
              >
                <Icon.Add />
              </Toolbar.Action>
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
            onFetchMore={fetchMore}
            replaceOnSingle
          >
            <List.Scroll full="y" onContextMenu={menuProps.open}>
              <List.Items<library.Key, library.Library>
                emptyContent={
                  answered && (
                    <Empty.Action
                      message="No libraries"
                      action={canCreate ? "Create library" : undefined}
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
  key: "library",
  icon: <Icon.Library />,
  content: <Content />,
  trigger: ["L"],
  tooltip: "Libraries",
  sizeBounds: { lower: 225, upper: 400 },
  initialSize: 300,
  useVisible: () => Access.useRetrieveGranted(library.TYPE_ONTOLOGY_ID),
};
