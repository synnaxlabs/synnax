// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu, Panel, type Triggers } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useTearOffTab } from "@/feature/panel/useTearOff";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { editTabName } from "@/platform/panel/tab";
import { Session } from "@/session";

const FOCUS_TRIGGER: Triggers.Trigger = ["Control", "L"];
const RENAME_TRIGGER: Triggers.Trigger = ["Control", "E"];

const RenameItem = (): ReactElement | null => {
  const tabKey = Panel.useTabKey();
  const isResource = Panel.useSelectTabVariant({}) === "resource";
  if (!isResource) return null;
  return (
    <CMenu.RenameItem
      onClick={() => editTabName(tabKey)}
      trigger={RENAME_TRIGGER}
      triggerIndicator
    />
  );
};

const FocusItem = (): ReactElement => {
  const tabKey = Panel.useTabKey();
  const startOverlaying = Session.Panel.useStartOverlaying();
  const handleFocus = useCallback(
    () => startOverlaying(tabKey),
    [startOverlaying, tabKey],
  );
  return (
    <Menu.Item itemKey="focus" onClick={handleFocus} trigger={FOCUS_TRIGGER}>
      <Icon.Focus />
      Focus
    </Menu.Item>
  );
};

const MoveToNewWindowItem = (): ReactElement => {
  const key = Panel.useKey();
  const tabKey = Panel.useTabKey();
  const getTab = Panel.useGetTab();
  const tearOff = useTearOffTab();
  const handleMove = useCallback(
    () => tearOff({ panel: key, tab: getTab({ key, tabKey }) }),
    [tearOff, key, tabKey, getTab],
  );
  return (
    <Menu.Item itemKey="move-to-new-window" onClick={handleMove}>
      <Icon.OpenInNewWindow />
      Move to New Window
    </Menu.Item>
  );
};

export const TabMenuItems = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  if (keys.length === 0) return <CMenu.ReloadConsoleItem />;
  return (
    <>
      <RenameItem />
      <FocusItem />
      <MoveToNewWindowItem />
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};
