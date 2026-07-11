// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import {
  createContext,
  memo,
  type ReactElement,
  type RefObject,
  useCallback,
  useContext,
  useMemo,
} from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import { useDispatch, useSelectLeafTabGroups, useSelectNode } from "@/panel/queries";
import { Scope, TabScope } from "@/panel/scope";
import { useKey } from "@/panel/Suspended";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";
import { Text } from "@/text";

export interface MosaicTabRenderProps {
  tabKey: string;
  visible: boolean;
}

export interface MosaicTabNameProps extends Omit<Tabs.NameProps, "value"> {
  onRename?: (tabKey: string, name: string) => void;
}

export interface MosaicProps extends Omit<
  Base.FrameProps,
  "onDrop" | "onCreate" | "onFileDrop" | "onResize" | "onSelect" | "children"
> {
  focused?: string;
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<MosaicTabRenderProps>;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
  onCreateTab?: () => panel.NewTab | undefined;
  resolveDroppedTab?: (key: string) => panel.NewTab | undefined;
}

// LeafContext carries the per-leaf render dependencies down to the granular Leaf nodes.
// Split nodes never read it, so a selection or focus change re-renders the leaf strips
// without touching the split tree.
interface LeafContextValue {
  portalRef: RefObject<Map<string, Portal.Node>>;
  preference: string[];
  focused?: string;
  onSelect?: (tabKey: string) => void;
  onClose: (tabKey: string) => void;
  onAdd: (path: number) => void;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
}

const LeafContext = createContext<LeafContextValue | null>(null);

const useLeafContext = (): LeafContextValue => {
  const ctx = useContext(LeafContext);
  if (ctx == null)
    throw new Error("[Panel.Mosaic] leaf rendered outside of a Mosaic frame");
  return ctx;
};

interface ContentProps extends Pick<MosaicProps, "children"> {
  tabKey: string;
  visible: boolean;
}

// Content publishes the tab scope around the consumer's render so it can read the tab's
// type/args with the granular tab-scoped selectors instead of receiving them as props.
const Content = ({ tabKey, visible, children }: ContentProps): ReactElement => (
  <TabScope.Provider value={tabKey}>{children({ tabKey, visible })}</TabScope.Provider>
);

export interface DefaultTabNameProps extends MosaicTabNameProps {
  name?: string;
  icon?: Icon.ReactElement;
}

// DefaultTabName renders a tab's name from the surrounding tab scope, so callers that
// don't supply a custom tabName get the standard name bound to the active tab. When an
// onRename handler is provided the name becomes editable and commits through it.
export const DefaultTabName = ({
  name,
  icon,
  onRename,
  level = "p",
  ...rest
}: DefaultTabNameProps): ReactElement => {
  const tabKey = TabScope.use();
  return (
    <>
      {icon != null && Icon.resolve(icon)}
      {onRename == null ? (
        <Text.Text level={level} overflow="ellipsis" {...rest}>
          {name}
        </Text.Text>
      ) : (
        <Text.Editable
          id={CSS.B(`tab-${tabKey}`)}
          level={level}
          value={name ?? ""}
          onChange={(next: string) => onRename(tabKey, next)}
          overflow="ellipsis"
          {...rest}
        />
      )}
    </>
  );
};

// resolveSelected picks the leaf's selected tab: the first preference present in the
// leaf's own tabs, falling back to the leaf's first tab.
const resolveSelected = (tabs: string[], preference: string[]): string | undefined =>
  preference.find((key) => tabs.includes(key)) ?? tabs[0];

const EMPTY_TABS: string[] = [];

interface LeafProps {
  path: number;
  tabs: string[];
}

// Leaf renders one mosaic leaf: its tab strip and the portal host for its selected tab.
// It is memoized on path and tab keys; a content change never reaches it, and a resize of
// another split never reaches it.
const Leaf = memo(({ path, tabs }: LeafProps): ReactElement => {
  const { portalRef, preference, focused, onSelect, onClose, onAdd, tabName } =
    useLeafContext();
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

// Node subscribes to the structural descriptor of a single tree position and renders
// either a split (with its two child Nodes) or a leaf. It is memoized on path alone, so a
// parent re-render never cascades; each Node re-renders only when its own node changes.
const Node = memo(({ path }: NodeProps): ReactElement | null => {
  const node = useSelectNode({ path });
  if (node == null) return null;
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
  return <Leaf path={path} tabs={node.tabs ?? EMPTY_TABS} />;
});
Node.displayName = "Panel.Mosaic.Node";

export const Mosaic = ({
  focused,
  selected,
  onSelect,
  children,
  tabName,
  onCreateTab,
  resolveDroppedTab,
  ...rest
}: MosaicProps): ReactElement | null => {
  const key = useKey();
  const { dispatch } = useDispatch();
  const groups = useSelectLeafTabGroups({ key });
  const preference = useMemo(() => selected ?? [], [selected]);

  const handleDrop = useCallback(
    ({ leafKey, tabKey, location, index }: Base.OnDropProps) =>
      dispatch({
        key,
        actions: [
          panel.moveTab({ key: tabKey, targetLeaf: Number(leafKey), index, location }),
        ],
      }),
    [dispatch, key],
  );

  const handleResize = useCallback(
    (splitKey: string, size: number) =>
      dispatch({
        key,
        actions: [panel.resizeSplit({ split: Number(splitKey), size })],
      }),
    [dispatch, key],
  );

  const handleClose = useCallback(
    (tabKey: string) => dispatch({ key, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, key],
  );

  const handleAdd = useCallback(
    (path: number) => {
      const tab = onCreateTab?.();
      if (tab == null) return;
      const action = panel.insertTab({ tab, targetLeaf: path });
      dispatch({ key, actions: [action] });
      if (action.type === "insert_tab") onSelect?.(action.insertTab.tab.key);
    },
    [dispatch, key, onSelect, onCreateTab],
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
      dispatch({ key, actions });
      const last = actions.at(-1);
      if (last?.type === "insert_tab") onSelect?.(last.insertTab.tab.key);
    },
    [dispatch, key, onSelect, resolveDroppedTab],
  );

  const tabKeys = useMemo(() => groups.flat(), [groups]);
  const visibleKeys = useMemo(() => {
    const visible = new Set<string>();
    groups.forEach((group) => {
      const sel = resolveSelected(group, preference);
      if (sel != null) visible.add(sel);
    });
    return visible;
  }, [groups, preference]);

  const [portalRef, portalNodes] = Portal.useNodes({
    keys: tabKeys,
    attrs: { style: "width: 100%; height: 100%; position: relative;" },
    onClick: onSelect,
    children: (tabKey) => (
      <Errors.Boundary>
        <Content tabKey={tabKey} visible={visibleKeys.has(tabKey)}>
          {children}
        </Content>
      </Errors.Boundary>
    ),
  });

  const ctx = useMemo<LeafContextValue>(
    () => ({
      portalRef,
      preference,
      focused,
      onSelect,
      onClose: handleClose,
      onAdd: handleAdd,
      tabName,
    }),
    [portalRef, preference, focused, onSelect, handleClose, handleAdd, tabName],
  );

  return (
    <Scope.Provider value={key}>
      {portalNodes}
      <LeafContext.Provider value={ctx}>
        <Base.Frame
          onDrop={handleDrop}
          onCreate={handleCreate}
          onResize={handleResize}
          {...rest}
        >
          <Node path={panel.ROOT_PATH} />
        </Base.Frame>
      </LeafContext.Provider>
    </Scope.Provider>
  );
};
