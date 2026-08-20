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
  Mosaic,
  Panel,
  Synnax,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import {
  createPillHaulItem,
  isPillHaulItem,
  PILL_HAUL_TYPE,
} from "@/feature/panel/haul";
import { useCreate } from "@/feature/panel/useCreate";
import { type Dwell, useDwell } from "@/feature/panel/useDwell";
import {
  type TabOrigin,
  useMoveTab,
  useMoveTabToNewPanel,
} from "@/feature/panel/useMoveTab";
import { OPEN_WINDOW_TRIGGER, useOpenWindow } from "@/feature/panel/useOpenWindow";
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
  const selected = Session.Panel.useSelectSelected();
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
      {keys.length === 1 && Session.Runtime.ENGINE === "tauri" && (
        <Menu.Item
          itemKey="open-in-new-window"
          onClick={() => openWindow(key)}
          triggerIndicator={key === selected ? OPEN_WINDOW_TRIGGER : undefined}
        >
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

// Only a tab dragged out of a mosaic can be dropped onto the strip. A pill dragged
// along the strip is a window gesture, resolved over the desktop.
const canDropTab: Haul.CanDrop = ({ items }) =>
  items.length === 1 && Mosaic.isTabDropHaulItem(items[0]);

interface TabDropReturn extends Haul.UseDropReturn {
  onDragLeave: () => void;
  /** Class marking the target as the one a dragged tab would land on. */
  className: string | false;
}

// The strip has no drop indicators of its own, so a target that accepts a tab has to
// say so itself.
const useTabDrop = (
  key: panel.Key | undefined,
  onDrop: (origin: TabOrigin) => void,
  onEnter?: () => void,
  enabled: boolean = true,
): TabDropReturn => {
  const [over, setOver] = useState(false);
  const handleDragOver = useCallback(() => {
    setOver(true);
    onEnter?.();
  }, [onEnter]);
  const handleDragLeave = useCallback(() => setOver(false), []);
  const canDrop = useCallback<Haul.CanDrop>(
    (props) => enabled && canDropTab(props),
    [enabled],
  );
  const { onDragOver, onDrop: handleDrop } = Haul.useDrop({
    type: "PanelSelector",
    key,
    canDrop,
    onDragOver: handleDragOver,
    onDrop: useCallback(
      ({ items }: Haul.OnDropProps) => {
        setOver(false);
        const origin = Panel.parseTabDragPayload(items[0]?.data);
        if (origin == null || origin.panel === key) return [];
        onDrop(origin);
        return items;
      },
      [key, onDrop],
    ),
  });
  return {
    onDragOver,
    onDrop: handleDrop,
    onDragLeave: handleDragLeave,
    className: over && CSS.M("tab-target"),
  };
};

interface TabProps {
  tabKey: panel.Key;
  dwell: Dwell;
}

// A pill whose panel vanished between the list answer and the retrieve
// renders nothing; the by-project subscription evicts the key right after.
const TabFallback = (): null => null;

const Tab = ({ tabKey, dwell }: TabProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null} FallbackComponent={TabFallback}>
    <TabContent tabKey={tabKey} dwell={dwell} />
  </Errors.SuspenseBoundary>
);

const TabContent = ({ tabKey, dwell }: TabProps): ReactElement => {
  Panel.useEnsure({ key: tabKey });
  const name = Panel.useName({ key: tabKey });
  const canEdit = Panel.useCanEdit({ key: tabKey });
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
  const moveTab = useMoveTab();
  const handleMove = useCallback(
    (origin: TabOrigin) => moveTab(origin, tabKey),
    [moveTab, tabKey],
  );
  const handleDwell = useCallback(() => dwell.enter(tabKey), [dwell, tabKey]);
  const { onDragLeave, className, ...dropProps } = useTabDrop(
    tabKey,
    handleMove,
    handleDwell,
    canEdit,
  );
  const handleDragLeave = useCallback(() => {
    onDragLeave();
    dwell.leave(tabKey);
  }, [onDragLeave, dwell, tabKey]);
  return (
    <Tabs.Tab
      itemKey={tabKey}
      className={CSS.cls(className)}
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragLeave={handleDragLeave}
      {...dropProps}
    >
      <Icon.Panel />
      <Text.MaybeEditable
        id={PCSS.B(`tab-${tabKey}`)}
        value={name}
        onChange={handleChange}
        disabled={!canEdit}
      />
    </Tabs.Tab>
  );
};

// The create button doubles as a drop target: releasing a tab on it mints a panel to
// hold the tab, the drag twin of the picker's "New panel" entry.
const CreateButton = (): ReactElement | null => {
  const selected = Session.Panel.useSelectSelected();
  const canCreate = Access.useCreateGranted(panel.TYPE_ONTOLOGY_ID);
  const handleCreate = useCreate();
  const moveToNewPanel = useMoveTabToNewPanel();
  const { className, ...dropProps } = useTabDrop(undefined, moveToNewPanel);
  if (!canCreate) return null;
  return (
    <Button.Button
      variant="text"
      textColor={9}
      className={CSS.cls(className)}
      onClick={handleCreate}
      {...dropProps}
    >
      <Icon.Add />
      {selected == null && "Create panel"}
    </Button.Button>
  );
};

const Internal = (): ReactElement => {
  const dispatch = useDispatch();
  const selected = Session.Panel.useSelectSelected();
  const ordered = Session.Panel.useSelectOrderedKeys();
  const dwell = useDwell();

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
            <Tab key={key} tabKey={key} dwell={dwell} />
          ))}
        </Tabs.Selector>
        <CreateButton />
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
