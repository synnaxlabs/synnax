// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import {
  Access,
  Button,
  CSS as PCSS,
  type Flux,
  Icon,
  type List,
  Menu,
  Panel,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/platform/context-menu";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<panel.Key, panel.Panel>;
}

const ContextMenu = ({ keys, getItem }: ContextMenuProps): ReactElement | null => {
  const ids = panel.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = Tree.useConfirmDelete({ type: "Panel" });
  const { update: del } = Panel.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<panel.Key | panel.Key[]>) => {
        const panelKeys = array.toArray(data);
        if (panelKeys.length === 0) return false;
        const panels = getItem(panelKeys);
        if (!(await confirm(panels))) return false;
        return data;
      },
      [getItem, confirm],
    ),
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {hasUpdatePermission && keys.length === 1 && (
        <>
          <CMenu.RenameItem onClick={() => Text.edit(PCSS.B(`tab-${key}`))} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
        <>
          <CMenu.DeleteItem onClick={() => del(keys)} />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};

interface TabProps {
  tabKey: panel.Key;
}

const Tab = ({ tabKey }: TabProps): ReactElement => {
  Panel.useEnsureRetrieved({ key: tabKey });
  const name = Panel.useSelectName({ key: tabKey });
  const { update: rename } = Panel.useRename();
  const handleChange = useCallback(
    (name: string) => rename({ key: tabKey, name }),
    [tabKey, rename],
  );
  return (
    <Tabs.Tab itemKey={tabKey}>
      <Text.Editable
        id={PCSS.B(`tab-${tabKey}`)}
        value={name}
        onChange={handleChange}
      />
    </Tabs.Tab>
  );
};

export const Selector = (): ReactElement | null => {
  const dispatch = useDispatch();
  const selected = Session.Panel.useSelectSelected();
  const { data, retrieve, getItem } = Panel.useList();
  useEffect(() => retrieve({}), [retrieve]);

  const handleSelect = useCallback(
    (key: string) => dispatch(Session.Panel.select({ key })),
    [dispatch],
  );

  const { update: create } = Panel.useCreate();
  const handleCreate = useCallback(() => create({ name: "New Panel" }), [create]);

  useEffect(() => {
    if (data.length === 0) return;
    if (selected != null && data.includes(selected)) return;
    dispatch(Session.Panel.select({ key: data[0] }));
  }, [selected, data, dispatch]);

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );
  const menuProps = Menu.useContextMenu();

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Tabs.Frame value={selected ?? ""} onChange={handleSelect}>
        <Tabs.Selector size="medium" variant="pill" onContextMenu={menuProps.open}>
          {data.map((key) => (
            <Tab key={key} tabKey={key} />
          ))}
          <Button.Button variant="text" sharp onClick={handleCreate}>
            <Icon.Add />
          </Button.Button>
        </Tabs.Selector>
      </Tabs.Frame>
    </Menu.ContextMenu>
  );
};
