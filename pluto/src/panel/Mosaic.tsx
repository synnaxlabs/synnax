// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type DragEventHandler, memo, type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import {
  useSelectLeafNode,
  useSelectNodeVariant,
  useSelectSelection,
  useSelectSplitNode,
  useSelectTabKeys,
  useSingleDispatch,
} from "@/panel/queries";
import { TabScope } from "@/panel/scope";
import { Portal } from "@/portal";
import { Select } from "@/select/base";
import { Tabs } from "@/tabs";

const PORTAL_NODE_ATTRS = {
  style: "width: 100%; height: 100%; position: relative;",
};

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
}

interface TabProps extends Pick<MosaicProps, "tabName"> {
  tabKey: string;
  onClose: (tabKey: string) => void;
}

const Tab = ({ tabKey, tabName, onClose }: TabProps): ReactElement => {
  const { startDrag, onDragEnd } = Base.useDragTab();
  const handleDragStart = useCallback<DragEventHandler<HTMLDivElement>>(
    (e) => startDrag(e, tabKey),
    [tabKey, startDrag],
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
}

const Leaf = memo(({ nodeKey, onAdd, ...rest }: NodeProps): ReactElement => {
  const { tabs } = useSelectLeafNode({ nodeKey });
  const selected = Select.useSelectedAmong(tabs) ?? tabs[0];
  const handleAdd = useCallback(() => onAdd(nodeKey), [nodeKey, onAdd]);
  return (
    <Base.Leaf nodeKey={nodeKey} grow>
      <Tabs.Frame grow>
        <Tabs.Selector>
          {tabs.map((tabKey) => (
            <Tab key={tabKey} tabKey={tabKey} {...rest} />
          ))}
          <Flex.Box grow />
          <Button.Button variant="text" sharp onClick={handleAdd}>
            <Icon.Add />
          </Button.Button>
        </Tabs.Selector>
        <Tabs.Content grow>
          <Portal.Out itemKey={selected} />
          <Base.Shield />
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
});
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

const EMPTY_SELECTED: string[] = [];

const PortalIn = memo(
  ({
    itemKey,
    onSelect,
    children,
  }: Pick<Portal.InProps, "itemKey"> &
    Pick<MosaicProps, "children" | "onSelect">): ReactElement => (
    <Portal.In itemKey={itemKey} attrs={PORTAL_NODE_ATTRS} onClick={onSelect}>
      <Errors.Boundary>
        <TabScope.Provider value={itemKey}>{children({})}</TabScope.Provider>
      </Errors.Boundary>
    </Portal.In>
  ),
);
PortalIn.displayName = "Panel.Mosaic.PortalIn";

const PortaledContents = memo(
  ({
    onSelect,
    children,
  }: Pick<MosaicProps, "onSelect" | "children">): ReactElement => {
    const keys = useSelectTabKeys();
    return (
      <>
        {keys.map((key) => (
          <PortalIn key={key} itemKey={key} onSelect={onSelect}>
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
  ...rest
}: MosaicProps): ReactElement | null => {
  const dispatch = useSingleDispatch();

  const handleDrop = useCallback(
    ({ nodeKey, tabKey, location, index }: Base.OnDropProps) =>
      dispatch(panel.moveTab({ key: tabKey, targetLeaf: nodeKey, index, location })),
    [dispatch],
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

  const selection = useSelectSelection({ selected });

  return (
    <Portal.Provider>
      <PortaledContents onSelect={onSelect}>{children}</PortaledContents>
      <Select.Context value={selection} onSelect={onSelect}>
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
            tabName={tabName}
          />
        </Base.Frame>
      </Select.Context>
    </Portal.Provider>
  );
};
