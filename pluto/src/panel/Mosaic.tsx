// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { memo, type ReactElement, type RefObject, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { context } from "@/context";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import {
  useSelectLeafTabGroups,
  useSelectNode,
  useSingleDispatch,
} from "@/panel/queries";
import { TabScope } from "@/panel/scope";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

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
  portalRef: RefObject<Map<string, Portal.Node>>;
  preference: string[];
  focused?: string;
  onSelect?: (tabKey: string) => void;
  onClose: (tabKey: string) => void;
  onAdd: (path: number) => void;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
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

interface LeafProps {
  path: number;
  tabs: string[];
}

// Leaf renders one mosaic leaf: its tab strip and the portal host for its selected tab.
// It is memoized on path and tab keys; a content change never reaches it, and a resize of
// another split never reaches it.
const Leaf = memo(({ path, tabs }: LeafProps): ReactElement => {
  const { portalRef, onClose, onAdd, tabName } = useContext("Panel.Leaf");
  const { startDrag, onDragEnd } = Base.useDragTab();
  const selected = resolveSelected(tabs, preference);
  const contentNode = selected != null ? portalRef.current.get(selected) : undefined;
  return (
    <Base.Leaf leafKey={path.toString()} grow>
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
          <Button.Button variant="text" sharp onClick={() => onAdd(path)}>
            <Icon.Add />
          </Button.Button>
        </Tabs.Selector>
        <Tabs.Content grow>
          {contentNode != null && <Portal.Out node={contentNode} />}
          <Base.Shield />
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
});
Leaf.displayName = "Panel.Mosaic.Leaf";

interface NodeProps {
  path: number;
}

const Node = memo(({ path }: NodeProps): ReactElement => {
  const node = useSelectNode({ path });
  if (node.variant === "split")
    return (
      <Base.Split
        splitKey={path.toString()}
        direction={node.direction}
        size={node.size}
      >
        <Node path={panel.childPath(path, "first")} />
        <Node path={panel.childPath(path, "last")} />
      </Base.Split>
    );
  return <Leaf path={path} tabs={node.tabs} />;
});
Node.displayName = "Panel.Mosaic.Node";

const EMPTY_SELECTED: string[] = [];

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
  const groups = useSelectLeafTabGroups();

  const handleDrop = useCallback(
    ({ leafKey, tabKey, location, index }: Base.OnDropProps) =>
      dispatch(
        panel.moveTab({ key: tabKey, targetLeaf: Number(leafKey), index, location }),
      ),
    [dispatch],
  );

  const handleResize = useCallback(
    (splitKey: string, size: number) =>
      dispatch(panel.resizeSplit({ split: Number(splitKey), size })),
    [dispatch],
  );

  const handleClose = useCallback(
    (tabKey: string) => dispatch(panel.removeTab({ key: tabKey })),
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
    ({ leafKey, location, tabKeys, index }: Base.OnCreateProps) => {
      const node = Number(leafKey);
      const tabs = tabKeys
        .map((tabKey) => resolveDroppedTab?.(tabKey))
        .filter((tab): tab is panel.NewTab => tab != null);
      if (tabs.length === 0) return;
      const restLeaf =
        location === "center" ? node : panel.childPath(node, panel.splitSide(location));
      const actions = tabs.map((tab, i) =>
        panel.insertTab(
          i === 0
            ? { tab, targetLeaf: node, location, index }
            : { tab, targetLeaf: restLeaf },
        ),
      );
      dispatch(actions);
      const last = actions.at(-1);
      if (last?.type === "insert_tab") onSelect?.(last.insertTab.tab.key);
    },
    [dispatch, onSelect, resolveDroppedTab],
  );

  const [portalRef, portalNodes] = Portal.useNodes({
    keys: tabKeys,
    attrs: { style: "width: 100%; height: 100%; position: relative;" },
    onClick: onSelect,
    children: (tabKey) => (
      <Errors.Boundary>
        <Content tabKey={tabKey}>{children}</Content>
      </Errors.Boundary>
    ),
  });

  const ctx = useMemo<ContextValue>(
    () => ({
      portalRef,
      onSelect,
      onClose: handleClose,
      onAdd: handleAdd,
      tabName,
    }),
    [portalRef, onSelect, handleClose, handleAdd, tabName],
  );

  return (
    <>
      {portalNodes}
      <Context value={ctx}>
        <Base.Frame
          onDrop={handleDrop}
          onCreate={handleCreate}
          onResize={handleResize}
          {...rest}
        >
          <Node path={panel.ROOT_PATH} />
        </Base.Frame>
      </Context>
    </>
  );
};
