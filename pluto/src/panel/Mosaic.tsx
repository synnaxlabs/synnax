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

interface LeafProps {
  path: number;
  tabs: string[];
}

// Leaf renders one mosaic leaf: its tab strip and the portal host for its selected tab.
// It is memoized on path and tab keys; a content change never reaches it, and a resize of
// another split never reaches it.
const Leaf = memo(({ path, tabs }: LeafProps): ReactElement => {
  const { preference, focused, onSelect, onClose, onAdd, tabName } =
    useContext("Panel.Leaf");
  const { startDrag, onDragEnd } = Base.useDragTab();
  const selected = resolveSelected(tabs, preference);
  return (
    <Base.Leaf leafKey={path} grow>
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
          {selected != null && <Portal.Out itemKey={selected} />}
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
      <Base.Split splitKey={path} direction={node.direction} size={node.size}>
        <Node path={panel.childPath(path, "first")} />
        <Node path={panel.childPath(path, "last")} />
      </Base.Split>
    );
  return <Leaf path={path} tabs={node.tabs} />;
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
    ({ leafKey, tabKey, location, index }: Base.OnDropProps) =>
      dispatch(panel.moveTab({ key: tabKey, targetLeaf: leafKey, index, location })),
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
          <Node path={panel.ROOT_PATH} />
        </Base.Frame>
      </Context>
    </Portal.Provider>
  );
};
