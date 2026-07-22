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
import { useDispatch } from "react-redux";

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
  const dispatch = useDispatch();
  const panelKey = Panel.useKey();
  const tabKey = Panel.useTabKey();
  const handleFocus = useCallback(
    () => dispatch(Session.Panel.startOverlaying({ key: panelKey, tabKey })),
    [dispatch, panelKey, tabKey],
  );
  return (
    <Menu.Item itemKey="focus" onClick={handleFocus} trigger={FOCUS_TRIGGER}>
      <Icon.Focus />
      Focus
    </Menu.Item>
  );
};

export const TabMenuItems = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  if (keys.length === 0) return <CMenu.ReloadConsoleItem />;
  return (
    <>
      <RenameItem />
      <FocusItem />
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};
