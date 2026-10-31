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
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Errors, type Flux, Mosaic, Panel, Synnax } from "@synnaxlabs/pluto";
import { array, record } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import {
  createPillHaulItem,
  isPillHaulItem,
  PILL_HAUL_TYPE,
} from "@/feature/panel/haul";
import { renderIcon } from "@/feature/panel/icon";
import { IconPicker, openIconPicker } from "@/feature/panel/IconPicker";
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

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  const ids = panel.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = Modals.useConfirmDelete({ type: "Panel" });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const openWindow = useOpenWindow();
  const selected = Session.Panel.useSelectSelected();
  const { update: del } = Panel.useDelete({
    beforeUpdate: async ({
      data,
    }: Flux.BeforeUpdateParams<panel.Key | panel.Key[]>) => {
      const panelKeys = array.toArray(data);
      if (panelKeys.length === 0) return false;
      const items = panelKeys.map((key) => {
        const cached = client?.panels.getCached(key);
        return { name: query.isLive(cached) ? cached.name : "this panel" };
      });
      if (!(await confirm(items))) return false;
      dispatch(Session.Panel.remove({ keys: panelKeys }));
      return data;
    },
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {hasUpdatePermission && keys.length === 1 && (
        <CMenu.RenameItem onClick={() => Text.edit(PCSS.B(`tab-${key}`))} />
      )}
      {hasUpdatePermission && keys.length === 1 && (
        <Menu.Item itemKey="change-icon" onClick={() => openIconPicker(key)}>
          <Icon.FillColor />
          Change icon
        </Menu.Item>
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

const contextMenu = Component.renderProp(ContextMenu);

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
  const handleDragOver = () => {
    setOver(true);
    onEnter?.();
  };
  const handleDragLeave = () => setOver(false);
  const canDrop: Haul.CanDrop = (props) => enabled && canDropTab(props);
  const { onDragOver, onDrop: handleDrop } = Haul.useDrop({
    type: "PanelSelector",
    key,
    canDrop,
    onDragOver: handleDragOver,
    onDrop: ({ items }: Haul.OnDropProps) => {
      setOver(false);
      const origin = Panel.parseTabDragPayload(items[0]?.data);
      if (origin == null || origin.panel === key) return [];
      onDrop(origin);
      return items;
    },
  });
  return {
    onDragOver,
    onDrop: handleDrop,
    onDragLeave: handleDragLeave,
    className: over && CSS.M("tab-target"),
  };
};

type AnchorHandler = (key: panel.Key, pill: HTMLElement | null) => void;

interface TabProps {
  tabKey: panel.Key;
  dwell: Dwell;
  onAnchorChange: AnchorHandler;
}

// A pill whose panel vanished between the list answer and the retrieve
// renders nothing; the by-project subscription evicts the key right after.
const TabFallback = (): null => null;

const Tab = (props: TabProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null} FallbackComponent={TabFallback}>
    <TabContent {...props} />
  </Errors.SuspenseBoundary>
);

const TabContent = ({ tabKey, dwell, onAnchorChange }: TabProps): ReactElement => {
  Panel.useEnsure({ key: tabKey });
  const name = Panel.useName({ key: tabKey });
  const icon = Panel.useIcon({ key: tabKey });
  const color = Panel.useColor({ key: tabKey });
  const canEdit = Panel.useCanEdit({ key: tabKey });
  const { update: rename } = Panel.useRename();
  const handleChange = (name: string) => rename({ key: tabKey, name });
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "PanelSelector" });
  const handleDragStart = () => startDrag([createPillHaulItem(tabKey)]);
  const moveTab = useMoveTab();
  const handleMove = (origin: TabOrigin) => moveTab(origin, tabKey);
  const handleDwell = () => dwell.enter(tabKey);
  const { onDragLeave, className, ...dropProps } = useTabDrop(
    tabKey,
    handleMove,
    handleDwell,
    canEdit,
  );
  const handleDragLeave = () => {
    onDragLeave();
    dwell.leave(tabKey);
  };
  const handleRef = (pill: HTMLDivElement | null) => onAnchorChange(tabKey, pill);
  return (
    <Tabs.Tab
      ref={handleRef}
      itemKey={tabKey}
      className={CSS.cls(className)}
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragLeave={handleDragLeave}
      {...dropProps}
    >
      {renderIcon({ icon, color })}
      <Text.MaybeEditable
        id={PCSS.B(`tab-${tabKey}`)}
        value={name}
        overflow="fade"
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

  const handleSelect = (key: string) => dispatch(Session.Panel.select({ key }));

  const handleDrop = ({ items, index }: Tabs.SelectorOnDropParams): Haul.Item[] => {
    const pills = items.filter(isPillHaulItem);
    if (pills.length > 0) dispatch(Session.Panel.reorder({ key: pills[0].key, index }));
    return pills;
  };

  const menuProps = Menu.useContextMenu();

  const [anchors, setAnchors] = useState<Record<panel.Key, HTMLElement>>({});
  const handleAnchorChange: AnchorHandler = (key, pill) =>
    setAnchors((anchors) =>
      pill == null ? record.omit(anchors, key) : { ...anchors, [key]: pill },
    );

  return (
    <>
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
              <Tab
                key={key}
                tabKey={key}
                dwell={dwell}
                onAnchorChange={handleAnchorChange}
              />
            ))}
          </Tabs.Selector>
          <CreateButton />
        </Tabs.Frame>
      </Menu.ContextMenu>
      {Object.entries(anchors).map(([key, anchor]) => (
        <IconPicker key={key} panel={key} anchor={anchor} />
      ))}
    </>
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
