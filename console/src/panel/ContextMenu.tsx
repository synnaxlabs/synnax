// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon, Menu, Panel as Base, Text } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";

export interface TabMenuItemsProps {
  panelKey: panel.Key;
  tabKey: string;
}

// TabMenuItems renders the standard context-menu items for a panel tab: rename,
// close, focus, and (when the tab's leaf holds more than one tab) splitting the
// leaf with this tab on the new side. All actions address the tab through the
// panel document; none touch the layout slice.
export const TabMenuItems = ({ panelKey, tabKey }: TabMenuItemsProps): ReactElement => {
  const dispatch = useDispatch();
  const windowKey = useSelectWindowKey();
  const { dispatch: dispatchPanel } = Base.useDispatch();
  const { data: p } = Base.useRetrieve({ key: panelKey });
  const leafPath = p != null ? panel.tabLeafPath(p.root, tabKey) : null;
  const leafTabCount =
    p != null && leafPath != null
      ? (panel.walkPath(p.root, leafPath)?.leaf?.tabs.length ?? 0)
      : 0;
  const canSplit = leafTabCount >= 2;
  const handleSplit = useCallback(
    (dir: direction.Direction) => {
      if (leafPath == null) return;
      const location = dir === "x" ? "right" : "bottom";
      dispatchPanel({
        key: panelKey,
        actions: [
          panel.splitLeaf({ leaf: leafPath, location, size: 0.5 }),
          panel.moveTab({
            key: tabKey,
            targetLeaf: panel.childPath(leafPath, "last"),
            index: 0,
          }),
        ],
      });
    },
    [dispatchPanel, panelKey, tabKey, leafPath],
  );
  const handleClose = useCallback(
    () => dispatchPanel({ key: panelKey, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatchPanel, panelKey, tabKey],
  );
  const handleFocus = useCallback(() => {
    if (windowKey != null) dispatch(Layout.setFocus({ windowKey, key: tabKey }));
  }, [dispatch, windowKey, tabKey]);
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
      <Menu.Item itemKey="focus" onClick={handleFocus} trigger={["Control", "L"]}>
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

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  panelKey: panel.Key;
}

// ContextMenu is the panel tab strip's context menu. The tab's content type can
// register a custom renderer (Layout.ContextMenuProvider) that replaces the whole
// menu, receiving the content key as layoutKey and the hosting tab's key;
// otherwise the standard tab items render.
export const ContextMenu = ({
  keys,
  panelKey,
}: ContextMenuProps): ReactElement | null => {
  const tabKey: string | undefined = keys[0];
  const { data: p } = Base.useRetrieve({ key: panelKey });
  const tab = p != null && tabKey != null ? panel.findTab(p.root, tabKey) : null;
  const type = tab?.resource?.type ?? tab?.view?.type ?? "";
  const Custom = Layout.useContextMenuRenderer(type);
  if (tabKey == null)
    return (
      <CMenu.Menu>
        <CMenu.ReloadConsoleItem />
      </CMenu.Menu>
    );
  if (tab == null) return null;
  if (Custom != null)
    return <Custom layoutKey={tab.resource?.key ?? tabKey} tabKey={tabKey} />;
  return (
    <CMenu.Menu>
      <TabMenuItems panelKey={panelKey} tabKey={tabKey} />
    </CMenu.Menu>
  );
};
