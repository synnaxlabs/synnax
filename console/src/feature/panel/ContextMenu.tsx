// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query } from "@synnaxlabs/client";
import { Access, Icon, Menu, Panel, Synnax, Triggers } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useMovePicker } from "@/feature/panel/MovePicker";
import { type TabOrigin } from "@/feature/panel/useMoveTab";
import { useTearOffTab } from "@/feature/panel/useTearOff";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { Panel as PlatformPanel } from "@/platform/panel";
import { Session } from "@/session";

// Split out because useTabResource throws on a view tab.
const ResourceRenameItem = (): ReactElement | null => {
  const tabKey = Panel.useTabKey();
  const resource = Panel.useTabResource({});
  const canRename = Access.useUpdateGranted(resource);
  if (!canRename) return null;
  return (
    <CMenu.RenameItem
      onClick={() => PlatformPanel.editTabName(tabKey)}
      trigger={PlatformPanel.RENAME_TRIGGER}
      triggerIndicator
    />
  );
};

const RenameItem = (): ReactElement | null =>
  Panel.useTabVariant({}) === "resource" ? <ResourceRenameItem /> : null;

const FocusItem = (): ReactElement => {
  const tabKey = Panel.useTabKey();
  const isOverlaid = Session.Panel.useSelectIsTabOverlaid();
  const startOverlaying = Session.Panel.useStartOverlaying();
  const dispatch = Session.useDispatch();
  const handleFocus = useCallback(() => {
    if (isOverlaid) dispatch(Session.Panel.stopOverlaying({}));
    else startOverlaying(tabKey);
  }, [isOverlaid, dispatch, startOverlaying, tabKey]);
  return (
    <Menu.Item
      itemKey="focus"
      onClick={handleFocus}
      triggerIndicator={isOverlaid ? Triggers.ESCAPE : Panel.OVERLAY_TRIGGER}
    >
      {isOverlaid ? <Icon.Collapse /> : <Icon.Focus />}
      {isOverlaid ? "Exit focus" : "Focus"}
    </Menu.Item>
  );
};

// Resolves the whole tab the menu is open on, which every move path needs: a tab
// travels to a panel that may never have loaded the source.
const useOrigin = (): (() => TabOrigin | undefined) => {
  const key = Panel.useKey();
  const tabKey = Panel.useTabKey();
  const client = Synnax.use();
  return useCallback(() => {
    const cached = client?.panels.getCached(key);
    if (!query.isLive(cached)) return undefined;
    const tab = panel.findTab(cached.root, tabKey);
    return tab == null ? undefined : { panel: key, tab };
  }, [key, tabKey, client]);
};

const SplitItems = (): ReactElement | null => {
  const isOverlaid = Session.Panel.useSelectIsTabOverlaid();
  if (isOverlaid) return null;
  return <Panel.SplitTabMenuItems />;
};

const MoveToPanelItem = (): ReactElement | null => {
  const getOrigin = useOrigin();
  const openPicker = useMovePicker();
  const canEdit = Panel.useCanEdit({});
  const handleMove = useCallback(() => {
    const origin = getOrigin();
    if (origin != null) openPicker({ origin });
  }, [getOrigin, openPicker]);
  if (!canEdit) return null;
  return (
    <Menu.Item itemKey="move-to-panel" onClick={handleMove}>
      <Icon.Panel />
      Move to panel
    </Menu.Item>
  );
};

const MoveToNewWindowItem = (): ReactElement | null => {
  const getOrigin = useOrigin();
  const tearOff = useTearOffTab();
  const canEdit = Panel.useCanEdit({});
  const canCreate = Access.useCreateGranted(panel.TYPE_ONTOLOGY_ID);
  const handleMove = useCallback(() => {
    const origin = getOrigin();
    if (origin != null) tearOff(origin);
  }, [getOrigin, tearOff]);
  if (!canEdit || !canCreate) return null;
  if (Session.Runtime.ENGINE !== "tauri") return null;
  return (
    <Menu.Item itemKey="move-to-new-window" onClick={handleMove}>
      <Icon.OpenInNewWindow />
      Move to new window
    </Menu.Item>
  );
};

export const TabMenuItems = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  if (keys.length === 0) return <CMenu.ReloadConsoleItem />;
  return (
    <>
      <Panel.CloseTabMenuItem />
      <Menu.Divider />
      <RenameItem />
      <FocusItem />
      <Menu.Divider />
      <SplitItems />
      <MoveToPanelItem />
      <MoveToNewWindowItem />
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};
