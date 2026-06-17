// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Icon, Menu, Panel as Base, Panel, Text } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/components";
import { Session } from "@/panel/session";

export interface TabMenuItemsProps {
  panelKey: panel.Key;
  tabKey: string;
}

export const TabMenuItems = ({ panelKey, tabKey }: TabMenuItemsProps): ReactElement => {
  const dispatch = useDispatch();
  const dispatchPanel = Base.useSingleDispatch();
  const { data: p } = Base.useRetrieve({ key: panelKey });
  const canSplit = p != null && panel.canSplitTab(p.root, tabKey);
  const handleSplit = useCallback(
    (direction: direction.Direction) =>
      dispatchPanel(panel.splitTab({ key: tabKey, direction })),
    [dispatchPanel, tabKey],
  );
  const handleClose = useCallback(
    () => dispatchPanel(panel.removeTab({ key: tabKey })),
    [dispatchPanel, panelKey, tabKey],
  );
  const handleOverlay = useCallback(() => {
    dispatch(Session.overlayTab({ tabKey }));
  }, [dispatch, tabKey]);
  return (
    <>
      <CMenu.RenameItem
        onClick={() => Text.edit(`pluto-tab-${tabKey}`)}
        trigger={["Control", "E"]}
        triggerIndicator
      />
      <Menu.Item
        itemKey="close"
        onClick={handleClose}
        trigger={["Control", "W"]}
        triggerIndicator
      >
        <Icon.Close />
        Close
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item itemKey="focus" onClick={handleOverlay} trigger={["Control", "L"]}>
        <Icon.Focus />
        Focus
      </Menu.Item>
      {canSplit && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="splitX" onClick={() => handleSplit("x")}>
            <Icon.SplitX />
            Split horizontally
          </Menu.Item>
          <Menu.Item itemKey="splitY" onClick={() => handleSplit("y")}>
            <Icon.SplitY />
            Split vertically
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {}

export const ContextMenu = ({ keys }: ContextMenuProps): ReactElement | null => {
  const panelKey = Panel.useKey("ContextMenu");
  const tabKey: string | undefined = keys[0];
  if (tabKey == null)
    return (
      <CMenu.Menu>
        <CMenu.ReloadConsoleItem />
      </CMenu.Menu>
    );
  return (
    <CMenu.Menu>
      <TabMenuItems panelKey={panelKey} tabKey={tabKey} />
    </CMenu.Menu>
  );
};
