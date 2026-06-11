// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, panel } from "@synnaxlabs/client";
import { type location, uuid } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { Mosaic as Base } from "@/mosaic";
import {
  type TabContent,
  useDispatch,
  useEnsureRetrieved,
  useSelectRoot,
  useSelectTab,
} from "@/panel/queries";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

export interface MosaicTabRenderProps extends TabContent {
  tabKey: string;
  visible: boolean;
}

export interface MosaicTabNameProps extends Tabs.NameProps, TabContent {}

export interface MosaicProps extends Omit<
  Base.MosaicProps,
  | "root"
  | "activeTab"
  | "children"
  | "Name"
  | "onDrop"
  | "onResize"
  | "onClose"
  | "onCreate"
  | "onSelect"
> {
  panelKey: panel.Key;
  focused?: string;
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<MosaicTabRenderProps>;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
}

const adaptToMosaic = (root: panel.Node, selected: string[] | undefined): Base.Node => {
  const preference = selected ?? [];
  const visit = (node: panel.Node | undefined, key: number): Base.Node => {
    if (node == null) return { key };
    if (node.split != null)
      return {
        key,
        direction: node.split.direction,
        size: node.split.size,
        first: visit(node.split.first, panel.childPath(key, "first")),
        last: visit(node.split.last, panel.childPath(key, "last")),
      };
    if (node.leaf == null) return { key, tabs: [] };
    const tabs: Tabs.Tab[] = node.leaf.tabs.map((t) => ({
      tabKey: t.key,
      name: "",
      closable: true,
      editable: t.resource != null || t.view != null,
    }));
    const selected =
      preference.find((key) => tabs.some((t) => t.tabKey === key)) ?? tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return visit(root, panel.ROOT_PATH);
};

interface TabNameContextValue {
  panelKey: panel.Key;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
}

const [TabNameContext, useTabNameContext] = context.create<TabNameContextValue>({
  displayName: "Panel.Mosaic.TabContext.Provider",
  providerName: "Panel.Mosaic.TabContext.Provider",
});

const TabName: ComponentType<Tabs.NameProps> = (props) => {
  const { tabKey } = props;
  const { panelKey: key, tabName } = useTabNameContext("Panel.Mosaic.TabName");
  const content = useSelectTab({ key, tabKey });
  if (tabName == null) return <Tabs.DefaultName {...props} />;
  return tabName({ ...props, ...content });
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
  children({ ...useSelectTab({ key, tabKey }), tabKey, visible });

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
  const treeRoot = useSelectRoot({ key });
  const root = useMemo(() => adaptToMosaic(treeRoot, selected), [treeRoot, selected]);

  const handleDrop = useCallback(
    (targetLeaf: number, tabKey: string, location: location.Location, index?: number) =>
      dispatch({
        key,
        actions: [panel.moveTab({ key: tabKey, targetLeaf, index, location })],
      }),
    [dispatch, key],
  );

  const handleResize = useCallback(
    (split: number, size: number) =>
      dispatch({ key, actions: [panel.resizeSplit({ split, size })] }),
    [dispatch, key],
  );

  const handleClose = useCallback(
    (tabKey: string) => dispatch({ key, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, key],
  );

  const handleCreate = useCallback(
    (node: number, loc: location.Location, tabKeys?: string[]) => {
      let tabs: panel.Tab[];
      if (tabKeys == null) tabs = [{ key: uuid.create() }];
      else
        tabs = tabKeys.flatMap((raw) => {
          const parsed = ontology.idZ.safeParse(raw);
          return parsed.success ? [{ key: uuid.create(), resource: parsed.data }] : [];
        });
      const restLeaf =
        loc === "center" ? node : panel.childPath(node, panel.splitSide(loc));
      const actions = tabs.map((tab, i) => {
        let payload = { tab, targetLeaf: restLeaf, location: undefined };
        if (i == 0) payload = { tab, targetLeaf: node, location: loc };
        return panel.insertTab(payload);
      });
      dispatch({ key, actions });
      onSelect?.(tabs[tabs.length - 1].key);
    },
    [dispatch, key, onSelect],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect,
    children: ({ tabKey, visible }) => (
      <Content panelKey={key} tabKey={tabKey} visible={visible !== false}>
        {children}
      </Content>
    ),
  });

  const tabNameContext = useMemo<TabNameContextValue>(
    () => ({ panelKey: key, tabName }),
    [key, tabName],
  );

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => {
      const node = portalRef.current.get(props.tabKey);
      if (node == null) return null;
      return <Portal.Out node={node} />;
    },
    [portalRef],
  );

  return (
    <TabNameContext value={tabNameContext}>
      {portalNodes}
      <Base.Mosaic
        {...rest}
        root={root}
        activeTab={focused}
        onSelect={onSelect}
        onDrop={handleDrop}
        onResize={handleResize}
        onClose={handleClose}
        onCreate={handleCreate}
        Name={tabName != null ? TabName : undefined}
      >
        {renderProp}
      </Base.Mosaic>
    </TabNameContext>
  );
};
