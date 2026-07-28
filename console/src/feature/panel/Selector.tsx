// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project } from "@synnaxlabs/client";
import {
  Access,
  Button,
  CSS as PCSS,
  Errors,
  type Flux,
  Haul,
  Icon,
  Menu,
  Panel,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { array, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { createPillHaulItem } from "@/feature/panel/haul";
import { useOpenWindow } from "@/feature/panel/useOpenWindow";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  panels: panel.Panel[];
}

const ContextMenu = ({ keys, panels }: ContextMenuProps): ReactElement | null => {
  const ids = panel.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = Tree.useConfirmDelete({ type: "Panel" });
  const dispatch = useDispatch();
  const openWindow = useOpenWindow();
  const { update: del } = Panel.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<panel.Key | panel.Key[]>) => {
        const panelKeys = array.toArray(data);
        if (panelKeys.length === 0) return false;
        const selected = panels.filter(({ key }) => panelKeys.includes(key));
        if (!(await confirm(selected))) return false;
        dispatch(Session.Panel.remove(panelKeys));
        return data;
      },
      [panels, confirm, dispatch],
    ),
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {keys.length === 1 && (
        <>
          <Menu.Item itemKey="open-in-new-window" onClick={() => openWindow(key)}>
            <Icon.OpenInNewWindow />
            Open in New Window
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
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

// A pill whose panel vanished between the list answer and the retrieve
// renders nothing; the by-project subscription evicts the key right after.
const TabFallback = (): null => null;

const Tab = ({ tabKey }: TabProps): ReactElement => (
  <Errors.SuspenseBoundary FallbackComponent={TabFallback}>
    <TabContent tabKey={tabKey} />
  </Errors.SuspenseBoundary>
);

const TabContent = ({ tabKey }: TabProps): ReactElement => {
  Panel.useEnsureRetrieved({ key: tabKey });
  const name = Panel.useSelectName({ key: tabKey });
  const { update: rename } = Panel.useRename();
  const handleChange = useCallback(
    (name: string) => rename({ key: tabKey, name }),
    [tabKey, rename],
  );
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "PanelSelector" });
  const handleDragStart = useCallback(
    () => startDrag([createPillHaulItem(tabKey)]),
    [startDrag, tabKey],
  );
  return (
    <Tabs.Tab
      itemKey={tabKey}
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
    >
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
  const projectKey = Session.Project.useSelectSelected();
  const { data } = Panel.useRetrieveByProject({ project: projectKey });
  const panels = useMemo(() => data ?? [], [data]);
  const keys = useMemo(() => panels.map(({ key }) => key), [panels]);

  const handleSelect = useCallback(
    (key: string) => dispatch(Session.Panel.select({ key })),
    [dispatch],
  );

  const { update: create } = Panel.useCreate();
  const handleCreate = useCallback(() => {
    const key = uuid.create();
    create({ key, name: "New Panel", parent: project.ontologyID(projectKey) });
    dispatch(Session.Panel.select({ key }));
  }, [create, dispatch, projectKey]);

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} panels={panels} />,
    [panels],
  );
  const menuProps = Menu.useContextMenu();

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Tabs.Frame value={selected ?? ""} onChange={handleSelect}>
        <Tabs.Selector
          className={CSS.B("panel-selector")}
          size="medium"
          variant="pill"
          onContextMenu={menuProps.open}
        >
          {keys.map((key) => (
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
