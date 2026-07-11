// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { memo, type ReactElement, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { context } from "@/context";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import { useSelectNode, useSelectTabKeys, useSingleDispatch } from "@/panel/queries";
import { TabScope } from "@/panel/scope";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

const PORTAL_NODE_ATTRS = {
  style: "width: 100%; height: 100%; position: relative;",
};

export interface MosaicProps extends Omit<
  Base.FrameProps,
  "onDrop" | "onCreate" | "onFileDrop" | "onResize" | "onSelect" | "children"
> {
  focused?: string;
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<{}>;
  tabName?: Component.RenderProp<{}>;
  onCreateTab?: () => panel.NewTab | undefined;
  resolveDroppedTab?: (key: string) => panel.NewTab | undefined;
}

interface ContextValue {
  preference: string[];
  focused?: string;
  onSelect?: (tabKey: string) => void;
  onClose: (tabKey: string) => void;
  onAdd: (path: number) => void;
  tabName?: Component.RenderProp<{}>;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "LeafContext",
  providerName: "LeafProvider",
});

interface ContentProps extends Pick<MosaicProps, "children"> {
  tabKey: string;
}

const Content = ({ tabKey, children }: ContentProps): ReactElement => (
  <TabScope.Provider value={tabKey}>{children({})}</TabScope.Provider>
);

// resolveSelected picks the leaf's selected tab: the first preference present in the
// leaf's own tabs, falling back to the leaf's first tab.
const resolveSelected = (tabs: string[], preference: string[]): string | undefined =>
  preference.find((key) => tabs.includes(key)) ?? tabs[0];

interface LeafProps extends Pick<Base.LeafProps, "nodeKey"> {
  tabs: string[];
}

// Leaf renders one mosaic leaf: its tab strip and the portal host for its selected tab.
// It is memoized on path and tab keys; a content change never reaches it, and a resize of
// another split never reaches it.
const Leaf = memo(({ nodeKey, tabs }: LeafProps): ReactElement => {
  const { preference, focused, onSelect, onClose, onAdd, tabName } =
    useContext("Panel.Leaf");
  const { startDrag, onDragEnd } = Base.useDragTab();
  const selected = resolveSelected(tabs, preference);
  return (
    <Base.Leaf nodeKey={nodeKey} grow>
      <Tabs.Frame value={selected} onChange={onSelect} onClose={onClose} grow>
        <Tabs.Selector altColor={focused != null && focused === selected}>
          {tabs.map((tabKey) => (
            <Tabs.Tab
              key={tabKey}
              itemKey={tabKey}
              draggable
              onDragStart={(e) => startDrag(e, tabKey)}
              onDragEnd={onDragEnd}
            >
              {tabName != null && (
                <TabScope.Provider value={tabKey}>{tabName({})}</TabScope.Provider>
              )}
              <Tabs.Close />
            </Tabs.Tab>
          ))}
          <Flex.Box grow />
          <Button.Button variant="text" sharp onClick={() => onAdd(nodeKey)}>
            <Icon.Add />
          </Button.Button>
        </Tabs.Selector>
        <Tabs.Content grow>
          {selected != null && <Portal.Out itemKey={selected} />}
          <Base.Shield />
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
});
Leaf.displayName = "Panel.Mosaic.Leaf";

interface NodeProps extends Pick<Base.SplitProps, "nodeKey" | "direction" | "size"> {}

const Split = memo(({ nodeKey, direction, size }: NodeProps): ReactElement => (
  <Base.Split nodeKey={nodeKey} direction={direction} size={size}>
    <Node nodeKey={panel.childNodeKey(nodeKey, "first")} />
    <Node nodeKey={panel.childNodeKey(nodeKey, "last")} />
  </Base.Split>
));
Split.displayName = "Panel.Mosaic.Split";

const Node = memo(({ nodeKey }: NodeProps): ReactElement => {
  const node = useSelectNode({ nodeKey });
  if (node.variant === "split")
    return <Split nodeKey={nodeKey} direction={node.direction} size={node.size} />;
  return <Leaf nodeKey={nodeKey} tabs={node.tabs} />;
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
        <Content tabKey={itemKey}>{children}</Content>
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
  focused,
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

  const ctx = useMemo<ContextValue>(
    () => ({
      preference: selected,
      focused,
      onSelect,
      onClose: handleClose,
      onAdd: handleAdd,
      tabName,
    }),
    [selected, focused, onSelect, handleClose, handleAdd, tabName],
  );

  return (
    <Portal.Provider>
      <PortaledContents onSelect={onSelect}>{children}</PortaledContents>
      <Context value={ctx}>
        <Base.Frame
          onDrop={handleDrop}
          onCreate={handleCreate}
          onResize={handleResize}
          {...rest}
        >
          <Node nodeKey={panel.ROOT_NODE_KEY} />
        </Base.Frame>
      </Context>
    </Portal.Provider>
  );
};
