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
      triggerIndicator={Panel.OVERLAY_TRIGGER}
    >
      {isOverlaid ? <Icon.Collapse /> : <Icon.Focus />}
      {isOverlaid ? "Exit focus" : "Focus"}
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
      <Panel.SplitTabMenuItems />
      <MoveToNewWindowItem />
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};
