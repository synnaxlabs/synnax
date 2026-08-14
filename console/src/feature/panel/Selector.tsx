// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/panel/Selector.css";

import { panel, query } from "@synnaxlabs/client";
import {
  Access,
  Button,
  type Component,
  CSS as PCSS,
  Errors,
  type Flux,
  Haul,
  Icon,
  Menu,
  Panel,
  Synnax,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import {
  createPillHaulItem,
  isPillHaulItem,
  PILL_HAUL_TYPE,
} from "@/feature/panel/haul";
import { useCreate } from "@/feature/panel/useCreate";
import { useOpenWindow } from "@/feature/panel/useOpenWindow";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  /** The strip's panels in render order, so a delete can hand the selection on. */
  order: panel.Key[];
}

const ContextMenu = ({ keys, order }: ContextMenuProps): ReactElement | null => {
  const ids = panel.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = Modals.useConfirmDelete({ type: "Panel" });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const openWindow = useOpenWindow();
  const { update: del } = Panel.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<panel.Key | panel.Key[]>) => {
        const panelKeys = array.toArray(data);
        if (panelKeys.length === 0) return false;
        const items = panelKeys.map((key) => {
          const cached = client?.panels.getCached(key);
          return { name: query.isLive(cached) ? cached.name : "this panel" };
        });
        if (!(await confirm(items))) return false;
        dispatch(Session.Panel.remove({ keys: panelKeys, order }));
        return data;
      },
      [client, confirm, dispatch, order],
    ),
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {hasUpdatePermission && keys.length === 1 && (
        <CMenu.RenameItem onClick={() => Text.edit(PCSS.B(`tab-${key}`))} />
      )}
      <Menu.Divider />
      {keys.length === 1 && (
        <Menu.Item itemKey="open-in-new-window" onClick={() => openWindow(key)}>
          <Icon.OpenInNewWindow />
          Open in new window
        </Menu.Item>
      )}
      <Menu.Divider />
      {hasDeletePermission && <CMenu.DeleteItem onClick={() => del(keys)} />}
      <Menu.Divider />
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
  <Errors.SuspenseBoundary loading={null} FallbackComponent={TabFallback}>
    <TabContent tabKey={tabKey} />
  </Errors.SuspenseBoundary>
);

const TabContent = ({ tabKey }: TabProps): ReactElement => {
  Panel.useEnsure({ key: tabKey });
  const name = Panel.useName({ key: tabKey });
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
      <Icon.Panel />
      <Text.Editable
        id={PCSS.B(`tab-${tabKey}`)}
        value={name}
        onChange={handleChange}
      />
    </Tabs.Tab>
  );
};

const Internal = (): ReactElement => {
  const dispatch = useDispatch();
  const selected = Session.Panel.useSelectSelected();
  const projectKey = Session.Project.useSelectSelected();
  const keys = Panel.useKeysByProject({ project: projectKey });
  const order = Session.Panel.useSelectOrder();
  // The query answers membership, the session answers order. A key the session
  // has not reconciled yet renders at the end in answer order: the sort is
  // stable and every unknown compares equal.
  const ordered = useMemo(() => {
    const slots = new Map(order.map((key, index) => [key, index]));
    return [...keys].sort(
      (a, b) => (slots.get(a) ?? order.length) - (slots.get(b) ?? order.length),
    );
  }, [keys, order]);

  const handleSelect = useCallback(
    (key: string) => dispatch(Session.Panel.select({ key })),
    [dispatch],
  );

  const handleDrop = useCallback(
    ({ items, index }: Tabs.SelectorOnDropParams): Haul.Item[] => {
      const pills = items.filter(isPillHaulItem);
      if (pills.length > 0)
        dispatch(Session.Panel.reorder({ key: pills[0].key, index }));
      return pills;
    },
    [dispatch],
  );

  const handleCreate = useCreate();
  const menuProps = Menu.useContextMenu();
  const contextMenu = useCallback<Component.RenderProp<Menu.ContextMenuMenuProps>>(
    (props) => <ContextMenu {...props} order={ordered} />,
    [ordered],
  );

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Tabs.Frame
        className={CSS.B("panel-selector")}
        value={selected ?? ""}
        onChange={handleSelect}
        x
        align="center"
        empty={false}
        gap="small"
      >
        <Tabs.Selector
          size="medium"
          variant="pill"
          haulType={PILL_HAUL_TYPE}
          onDrop={handleDrop}
          onContextMenu={menuProps.open}
        >
          {ordered.map((key) => (
            <Tab key={key} tabKey={key} />
          ))}
        </Tabs.Selector>
        <Button.Button variant="text" textColor={9} onClick={handleCreate}>
          <Icon.Add />
          {selected == null && "New Panel"}
        </Button.Button>
      </Tabs.Frame>
    </Menu.ContextMenu>
  );
};

// The strip is nav chrome with no room for a diagnostic, and a disconnected
// client throws straight out of the query, so every failure shows no strip.
const SelectorFallback = (): null => null;

export const Selector = (): ReactElement => (
  <Errors.SuspenseBoundary FallbackComponent={SelectorFallback}>
    <Internal />
  </Errors.SuspenseBoundary>
);
