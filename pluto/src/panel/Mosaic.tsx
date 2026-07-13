// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, panel } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Mosaic as Base } from "@/mosaic";
import {
  type TabContent,
  tabContent,
  useDispatch,
  useEnsureRetrieved,
  useSelectRoot,
  useSelectTab,
} from "@/panel/queries";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

// Portaled tab content fills its hosting Out, which anchors the absolutely
// positioned internals of visualizations rendered inside it.
const PORTAL_OUT_STYLE: CSSProperties = {
  width: "100%",
  height: "100%",
  position: "relative",
};

export interface MosaicTabRenderProps extends TabContent {
  tabKey: string;
  visible: boolean;
}

export interface MosaicTabNameProps extends TabContent {
  tabKey: string;
}

export interface MosaicProps extends Omit<
  Base.FrameProps,
  "onDrop" | "onCreate" | "onFileDrop" | "onResize" | "onSelect" | "children"
> {
  panelKey: panel.Key;
  focused?: string;
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<MosaicTabRenderProps>;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
}

interface TabNameProps {
  panelKey: panel.Key;
  tabKey: string;
  tabName: Component.RenderProp<MosaicTabNameProps>;
}

// TabName must stay module-level: the leaves below only re-parameterize it, so
// React updates name nodes in place. A component type created per render would
// remount every name node, dropping in-flight edit state.
const TabName = ({
  panelKey: key,
  tabKey,
  tabName,
}: TabNameProps): ReactElement | null => {
  const content = useSelectTab({ key, tabKey });
  return tabName({ tabKey, ...tabContent(content) });
};

interface ContentProps extends Pick<MosaicProps, "children"> {
  panelKey: panel.Key;
  tabKey: string;
  visible: boolean;
}

const Content = ({
  panelKey: key,
  tabKey,
  visible,
  children,
}: ContentProps): ReactElement | null =>
  children({ ...tabContent(useSelectTab({ key, tabKey })), tabKey, visible });

// resolveSelected picks the leaf's selected tab: the first preference present in
// the leaf's own tabs, falling back to the leaf's first tab.
const resolveSelected = (tabs: panel.Tab[], preference: string[]): string | undefined =>
  preference.find((key) => tabs.some((t) => t.key === key)) ?? tabs[0]?.key;

interface LeafProps extends Pick<MosaicProps, "focused" | "onSelect" | "tabName"> {
  panelKey: panel.Key;
  path: number;
  tabs: panel.Tab[];
  selected?: string;
  onClose: (tabKey: string) => void;
  onAdd: (path: number) => void;
}

const Leaf = ({
  panelKey,
  path,
  tabs,
  selected,
  focused,
  onSelect,
  onClose,
  onAdd,
  tabName,
}: LeafProps): ReactElement => {
  const { startDrag, onDragEnd } = Base.useDragTab();
  const tabKeys = useMemo(() => tabs.map(({ key }) => key), [tabs]);
  const selectorDropProps = Base.useSelectorDropProps({ nodeKey: path, tabKeys });
  return (
    <Base.Leaf nodeKey={path} grow>
      <Tabs.Frame value={selected} onChange={onSelect} grow>
        <Tabs.Selector {...selectorDropProps}>
          {tabs.map(({ key }) => (
            <Tabs.Tab
              key={key}
              itemKey={key}
              className={CSS.altColor(focused != null && focused === key) || undefined}
              draggable
              onDragStart={(e) => startDrag(e, key)}
              onDragEnd={onDragEnd}
            >
              {tabName != null && (
                <TabName panelKey={panelKey} tabKey={key} tabName={tabName} />
              )}
              <Tabs.Close onClick={() => onClose(key)} />
            </Tabs.Tab>
          ))}
          <Flex.Box grow />
          <Button.Button variant="text" sharp onClick={() => onAdd(path)}>
            <Icon.Add />
          </Button.Button>
        </Tabs.Selector>
        <Tabs.Content grow>
          {selected != null && (
            <Portal.Out itemKey={selected} style={PORTAL_OUT_STYLE} />
          )}
          <Base.Shield />
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
};

export const Mosaic = ({
  panelKey: key,
  focused,
  selected,
  onSelect,
  children,
  tabName,
  ...rest
}: MosaicProps): ReactElement | null => {
  useEnsureRetrieved({ key });
  const { dispatch } = useDispatch();
  const root = useSelectRoot({ key });
  const preference = useMemo(() => selected ?? [], [selected]);

  const handleDrop = useCallback(
    ({ nodeKey, tabKey, location, index }: Base.OnDropProps) =>
      dispatch({
        key,
        actions: [panel.moveTab({ key: tabKey, targetLeaf: nodeKey, index, location })],
      }),
    [dispatch, key],
  );

  const handleResize = useCallback(
    (splitKey: number, size: number) =>
      dispatch({ key, actions: [panel.resizeSplit({ split: splitKey, size })] }),
    [dispatch, key],
  );

  const handleClose = useCallback(
    (tabKey: string) => dispatch({ key, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, key],
  );

  const handleAdd = useCallback(
    (path: number) => {
      const tab: panel.Tab = { variant: "empty", key: uuid.create() };
      dispatch({ key, actions: [panel.insertTab({ tab, targetLeaf: path })] });
      onSelect?.(tab.key);
    },
    [dispatch, key, onSelect],
  );

  const handleCreate = useCallback(
    ({ nodeKey, location, tabKeys, index }: Base.OnCreateProps) => {
      const tabs = tabKeys.flatMap((raw): panel.Tab[] => {
        const parsed = ontology.idZ.safeParse(raw);
        return parsed.success
          ? [{ variant: "resource", key: uuid.create(), resource: parsed.data }]
          : [];
      });
      if (tabs.length === 0) return;
      const restLeaf =
        location === "center"
          ? nodeKey
          : panel.childPath(nodeKey, panel.splitSide(location));
      const actions = tabs.map((tab, i) => {
        let payload: panel.InsertTabPayload = { tab, targetLeaf: restLeaf };
        if (i === 0) payload = { tab, targetLeaf: nodeKey, location, index };
        return panel.insertTab(payload);
      });
      dispatch({ key, actions });
      onSelect?.(tabs[tabs.length - 1].key);
    },
    [dispatch, key, onSelect],
  );

  // One traversal derives both portal enumeration inputs: every tab key in the
  // tree and the set of keys visible as their leaf's resolved selection.
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

  const renderNode = (node: panel.Node, path: number): ReactElement => {
    if (node.variant === "split")
      return (
        <Base.Split
          key={path}
          nodeKey={path}
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
        panelKey={key}
        path={path}
        tabs={node.tabs}
        selected={sel}
        focused={focused}
        onSelect={onSelect}
        onClose={handleClose}
        onAdd={handleAdd}
        tabName={tabName}
      />
    );
  };

  // Content renders into portaled elements hosted from each leaf, so moving a
  // tab around the mosaic does not remount it: the leaf's Out re-parents the
  // same element, preserving DOM state and expensive resources like WebGL
  // contexts.
  return (
    <Portal.Context>
      {tabKeys.map((tabKey) => (
        <Portal.In key={tabKey} itemKey={tabKey} onClick={onSelect}>
          <Errors.Boundary>
            <Content panelKey={key} tabKey={tabKey} visible={visibleKeys.has(tabKey)}>
              {children}
            </Content>
          </Errors.Boundary>
        </Portal.In>
      ))}
      <Base.Frame
        onDrop={handleDrop}
        onCreate={handleCreate}
        onResize={handleResize}
        {...rest}
      >
        {renderNode(root, panel.ROOT_PATH)}
      </Base.Frame>
    </Portal.Context>
  );
};
