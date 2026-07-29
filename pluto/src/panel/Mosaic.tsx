// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/panel/Mosaic.css";

import { panel } from "@synnaxlabs/client";
import { type direction } from "@synnaxlabs/x";
import { type DragEventHandler, memo, type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Mosaic as Base } from "@/mosaic";
import { createTabDragPayload, parseTabDragPayload } from "@/panel/haul";
import {
  useDispatch,
  useGetTab,
  useSelectLeafNode,
  useSelectNodeVariant,
  useSelectRoot,
  useSelectSplitNode,
  useSelectTabKeys,
  useSingleDispatch,
} from "@/panel/queries";
import { Scope, TabScope } from "@/panel/scope";
import { Portal } from "@/portal";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { type Triggers } from "@/triggers";

export interface MosaicProps extends Omit<
  Base.FrameProps,
  "onDrop" | "onCreate" | "onFileDrop" | "onResize" | "onSelect" | "children"
> {
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<{}>;
  tabName?: Component.RenderProp<{}>;
  onCreateTab?: () => panel.NewTab | undefined;
  resolveDroppedTab?: (key: string) => panel.NewTab | undefined;
  extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
}

interface TabProps extends Pick<MosaicProps, "tabName"> {
  tabKey: string;
  onClose: (tabKey: string) => void;
}

const Tab = ({ tabKey, tabName, onClose }: TabProps): ReactElement => {
  const { startDrag, onDragEnd } = Base.useDragTab();
  const key = Scope.use();
  const getTab = useGetTab();
  const handleDragStart = useCallback<DragEventHandler<HTMLDivElement>>(
    (e) => startDrag(e, tabKey, createTabDragPayload(key, getTab({ tabKey }))),
    [tabKey, startDrag, key, getTab],
  );
  const handleClose = useCallback(() => onClose(tabKey), [tabKey, onClose]);
  return (
    <Tabs.Tab
      key={tabKey}
      itemKey={tabKey}
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
    >
      <TabScope.Provider value={tabKey}>{tabName?.({})}</TabScope.Provider>
      <Tabs.Close onClick={handleClose} />
    </Tabs.Tab>
  );
};

interface NodeProps
  extends Pick<Base.LeafProps, "nodeKey">, Pick<TabProps, "tabName" | "onClose"> {
  onAdd: (nodeKey: number) => void;
  onContextMenu: Menu.ContextMenuOpen;
}

const Leaf = memo(
  ({ nodeKey, onAdd, onContextMenu, ...rest }: NodeProps): ReactElement => {
    const { tabs } = useSelectLeafNode({ nodeKey });
    const selected = Select.useSelectedAmong(tabs) ?? tabs[0];
    const { onSelect } = Select.useContext();
    const handleAdd = useCallback(() => onAdd(nodeKey), [nodeKey, onAdd]);
    const handleSelectContent = useCallback(
      () => onSelect(selected),
      [onSelect, selected],
    );
    const selectorDropProps = Base.useSelectorDropProps({ nodeKey, tabKeys: tabs });
    return (
      <Base.Leaf nodeKey={nodeKey} grow>
        <Tabs.Frame grow>
          <Tabs.Selector {...selectorDropProps} onContextMenu={onContextMenu}>
            {tabs.map((tabKey) => (
              <Tab key={tabKey} tabKey={tabKey} {...rest} />
            ))}
            <Flex.Box grow />
            <Button.Button variant="text" onClick={handleAdd}>
              <Icon.Add color={10} />
            </Button.Button>
          </Tabs.Selector>
          <Tabs.Content grow>
            {selected != null && (
              <Portal.Out
                itemKey={selected}
                className={CSS.BE("panel-mosaic", "portal-out")}
                onClickCapture={handleSelectContent}
              />
            )}
            <Base.Shield />
          </Tabs.Content>
        </Tabs.Frame>
      </Base.Leaf>
    );
  },
);
Leaf.displayName = "Panel.Mosaic.Leaf";

const Split = memo(({ nodeKey, ...rest }: NodeProps): ReactElement => {
  const { direction, size } = useSelectSplitNode({ nodeKey });
  return (
    <Base.Split nodeKey={nodeKey} direction={direction} size={size}>
      <Node nodeKey={panel.childNodeKey(nodeKey, "first")} {...rest} />
      <Node nodeKey={panel.childNodeKey(nodeKey, "last")} {...rest} />
    </Base.Split>
  );
});
Split.displayName = "Panel.Mosaic.Split";

const Node = memo(({ nodeKey, ...rest }: NodeProps): ReactElement => {
  const C = useSelectNodeVariant({ nodeKey }) == "split" ? Split : Leaf;
  return <C nodeKey={nodeKey} {...rest} />;
});
Node.displayName = "Panel.Mosaic.Node";

const CLOSE_TRIGGER: Triggers.Trigger = ["Control", "W"];

interface TabMenuItemsProps {
  tabKey: string;
}

const TabMenuItems = ({ tabKey }: TabMenuItemsProps): ReactElement => {
  const dispatch = useSingleDispatch();
  const root = useSelectRoot({});
  const handleClose = useCallback(
    () => dispatch(panel.removeTab({ key: tabKey })),
    [dispatch, tabKey],
  );
  const handleSplit = useCallback(
    (direction: direction.Direction) =>
      dispatch(panel.splitTab({ key: tabKey, direction })),
    [dispatch, tabKey],
  );
  return (
    <>
      <Menu.Item
        itemKey="close"
        onClick={handleClose}
        trigger={CLOSE_TRIGGER}
        triggerIndicator
      >
        <Icon.Close />
        Close
      </Menu.Item>
      {panel.canSplitTab(root, tabKey) && (
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
    </>
  );
};

const EMPTY_SELECTED: string[] = [];

const PortalIn = memo(
  ({
    itemKey,
    children,
  }: Pick<Portal.InProps, "itemKey"> & Pick<MosaicProps, "children">): ReactElement => (
    <Portal.In itemKey={itemKey}>
      <Errors.Boundary>
        <TabScope.Provider value={itemKey}>{children({})}</TabScope.Provider>
      </Errors.Boundary>
    </Portal.In>
  ),
);
PortalIn.displayName = "Panel.Mosaic.PortalIn";

// Content renders into portaled elements hosted from each leaf, so moving a tab
// around the mosaic does not remount it: the destination leaf's Out re-parents
// the same element, preserving DOM state and expensive resources like WebGL
// contexts.
const PortaledContents = memo(
  ({ children }: Pick<MosaicProps, "children">): ReactElement => {
    const keys = useSelectTabKeys();
    return (
      <>
        {keys.map((key) => (
          <PortalIn key={key} itemKey={key}>
            {children}
          </PortalIn>
        ))}
      </>
    );
  },
);
PortaledContents.displayName = "Panel.Mosaic.PortaledContents";

export const Mosaic = ({
  selected = EMPTY_SELECTED,
  onSelect,
  children,
  tabName,
  onCreateTab,
  resolveDroppedTab,
  extraMenuItems,
  ...rest
}: MosaicProps): ReactElement | null => {
  const dispatch = useSingleDispatch();
  const key = Scope.use();
  const { dispatch: dispatchTo } = useDispatch();

  // A tab dropped from another panel is removed there and inserted here: the two
  // panels are separate documents, so the move is two dispatches (see the MoveTab
  // contract in the panel schema). The insert runs first so a failed remove leaves
  // the tab in both panels rather than nowhere.
  const handleDrop = useCallback(
    ({ nodeKey, tabKey, location, index, data }: Base.OnDropProps) => {
      const source = parseTabDragPayload(data);
      if (source == null || source.panel === key) {
        dispatch(panel.moveTab({ key: tabKey, targetLeaf: nodeKey, index, location }));
        return;
      }
      dispatchTo({
        key,
        actions: panel.insertTab({
          tab: source.tab,
          targetLeaf: nodeKey,
          index,
          location,
        }),
      });
      dispatchTo({ key: source.panel, actions: panel.removeTab({ key: tabKey }) });
      onSelect?.(tabKey);
    },
    [dispatch, dispatchTo, key, onSelect],
  );

  const handleResize = useCallback(
    (split: number, size: number) => dispatch(panel.resizeSplit({ split, size })),
    [dispatch],
  );

  const handleClose = useCallback(
    (key: string) => dispatch(panel.removeTab({ key })),
    [dispatch],
  );

  const handleAdd = useCallback(
    (path: number) => {
      const tab = onCreateTab?.();
      if (tab == null) return;
      const action = panel.insertTab({ tab, targetLeaf: path });
      dispatch(action);
      if (action.type === "insert_tab") onSelect?.(action.insertTab.tab.key);
    },
    [dispatch, onSelect, onCreateTab],
  );

  const handleCreate = useCallback(
    ({ nodeKey, location, tabKeys, index }: Base.OnCreateProps) => {
      const tabs = tabKeys
        .map((tabKey) => resolveDroppedTab?.(tabKey))
        .filter((tab): tab is panel.NewTab => tab != null);
      if (tabs.length === 0) return;
      const restLeaf =
        location === "center"
          ? nodeKey
          : panel.childNodeKey(nodeKey, panel.splitSide(location));
      const actions = tabs.map((tab, i) =>
        panel.insertTab(
          i === 0
            ? { tab, targetLeaf: nodeKey, location, index }
            : { tab, targetLeaf: restLeaf },
        ),
      );
      dispatch(actions);
      const last = actions.at(-1);
      if (last?.type === "insert_tab") onSelect?.(last.insertTab.tab.key);
    },
    [dispatch, onSelect, resolveDroppedTab],
  );

  const menuProps = Menu.useContextMenu();
  const renderMenu = useCallback<Component.RenderProp<Menu.ContextMenuMenuProps>>(
    (props) => {
      const tabKey: string | undefined = props.keys[0];
      if (tabKey == null)
        return (
          <Menu.Menu level="small" gap="small">
            {extraMenuItems?.(props)}
          </Menu.Menu>
        );
      return (
        <TabScope.Provider value={tabKey}>
          <Menu.Menu level="small" gap="small">
            <TabMenuItems tabKey={tabKey} />
            {extraMenuItems != null && <Menu.Divider />}
            {extraMenuItems?.(props)}
          </Menu.Menu>
        </TabScope.Provider>
      );
    },
    [extraMenuItems],
  );

  return (
    <Portal.Context>
      <PortaledContents>{children}</PortaledContents>
      <Select.Context value={selected} onSelect={onSelect}>
        <Menu.ContextMenu menu={renderMenu} {...menuProps}>
          <Base.Frame
            onDrop={handleDrop}
            onCreate={handleCreate}
            onResize={handleResize}
            {...rest}
          >
            <Node
              nodeKey={panel.ROOT_NODE_KEY}
              onClose={handleClose}
              onAdd={handleAdd}
              onContextMenu={menuProps.open}
              tabName={tabName}
            />
          </Base.Frame>
        </Menu.ContextMenu>
      </Select.Context>
    </Portal.Context>
  );
};
