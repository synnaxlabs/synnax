// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type ReactElement, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import { useDispatch, useSelectRoot } from "@/panel/queries";
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

// resolveSelected picks the leaf's selected tab: the first preference present in
// the leaf's own tabs, falling back to the leaf's first tab.
const resolveSelected = (tabs: panel.Tab[], preference: string[]): string | undefined =>
  preference.find((key) => tabs.some((t) => t.key === key)) ?? tabs[0]?.key;

interface LeafProps extends Pick<MosaicProps, "focused" | "onSelect" | "tabName"> {
  path: number;
  tabs: panel.Tab[];
  selected?: string;
  onClose: (tabKey: string) => void;
  onAdd: (path: number) => void;
  contentNode?: Portal.Node;
}

const Leaf = ({
  path,
  tabs,
  selected,
  focused,
  onSelect,
  onClose,
  onAdd,
  tabName,
  contentNode,
}: LeafProps): ReactElement => {
  const { startDrag, onDragEnd } = Base.useDragTab();
  return (
    <Base.Leaf leafKey={path.toString()} grow>
      <Tabs.Frame value={selected} onChange={onSelect} onClose={onClose} grow>
        <Tabs.Selector altColor={focused != null && focused === selected}>
          {tabs.map(({ key }) => (
            <Tabs.Tab
              key={key}
              itemKey={key}
              draggable
              onDragStart={(e) => startDrag(e, key)}
              onDragEnd={onDragEnd}
            >
              {tabName != null && (
                <TabScope.Provider value={key}>{tabName({})}</TabScope.Provider>
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
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
};

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
  const root = useSelectRoot({ key });
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

  // One traversal derives both portal enumeration inputs: every tab key in the tree
  // and the set of keys visible as their leaf's resolved selection.
  const [tabKeys, visibleKeys] = useMemo(() => {
    const keys: string[] = [];
    const visible = new Set<string>();
    const visit = (node: panel.Node): void => {
      if (node.variant === "split") {
        visit(node.first);
        visit(node.last);
        return;
      }
      node.tabs.forEach((t) => keys.push(t.key));
      const sel = resolveSelected(node.tabs, preference);
      if (sel != null) visible.add(sel);
    };
    visit(root);
    return [keys, visible];
  }, [root, preference]);

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

  const renderNode = (node: panel.Node, path: number): ReactElement => {
    if (node.variant === "split")
      return (
        <Base.Split
          key={path}
          splitKey={path.toString()}
          direction={node.direction}
          size={node.size}
        >
          {renderNode(node.first, panel.childPath(path, "first"))}
          {renderNode(node.last, panel.childPath(path, "last"))}
        </Base.Split>
      );
    const sel = resolveSelected(node.tabs, preference);
    return (
      <Leaf
        key={path}
        path={path}
        tabs={node.tabs}
        selected={sel}
        focused={focused}
        onSelect={onSelect}
        onClose={handleClose}
        onAdd={handleAdd}
        tabName={tabName}
        contentNode={sel != null ? portalRef.current.get(sel) : undefined}
      />
    );
  };

  return (
    <Scope.Provider value={key}>
      {portalNodes}
      <Base.Frame
        onDrop={handleDrop}
        onCreate={handleCreate}
        onResize={handleResize}
        {...rest}
      >
        {renderNode(root, panel.ROOT_PATH)}
      </Base.Frame>
    </Scope.Provider>
  );
};
