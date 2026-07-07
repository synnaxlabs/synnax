// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { type Component } from "@/component";
import { Mosaic as Base } from "@/mosaic";
import { useDispatch, useSelectStructure } from "@/panel/queries";
import { Scope, TabScope } from "@/panel/scope";
import { useKey } from "@/panel/Suspended";
import { Portal } from "@/portal";
import { Tabs } from "@/tabs";

export interface MosaicTabRenderProps {
  tabKey: string;
  visible: boolean;
}

export interface MosaicTabNameProps extends Omit<Tabs.NameProps, "tabKey"> {}

export interface MosaicProps extends Omit<
  Base.MosaicProps,
  | "root"
  | "activeTab"
  | "children"
  | "tabName"
  | "onDrop"
  | "onResize"
  | "onClose"
  | "onCreate"
  | "onSelect"
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

export interface DefaultTabNameProps extends Omit<Tabs.DefaultNameProps, "tabKey"> {}

// DefaultTabName renders a tab's name from the surrounding tab scope, so callers that
// don't supply a custom tabName get the standard editable name bound to the active tab.
export const DefaultTabName = (props: DefaultTabNameProps): ReactElement => {
  const tabKey = TabScope.use();
  return <Tabs.DefaultName tabKey={tabKey} {...props} />;
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
  const root = useSelectStructure({ key, selected });

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
    (node: number, location: location.Location, tabKeys?: string[]) => {
      const tabs = (
        tabKeys == null
          ? [onCreateTab?.()]
          : tabKeys.map((tabKey) => resolveDroppedTab?.(tabKey))
      ).filter((tab): tab is panel.NewTab => tab != null);
      if (tabs.length === 0) return;
      const restLeaf =
        location === "center" ? node : panel.childPath(node, panel.splitSide(location));
      const actions = tabs.map((tab, i) =>
        panel.insertTab(
          i === 0 ? { tab, targetLeaf: node, location } : { tab, targetLeaf: restLeaf },
        ),
      );
      dispatch({ key, actions });
      const last = actions.at(-1);
      if (last?.type === "insert_tab") onSelect?.(last.insertTab.tab.key);
    },
    [dispatch, key, onSelect, onCreateTab, resolveDroppedTab],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect,
    children: ({ tabKey, visible }) => (
      <Content tabKey={tabKey} visible={visible !== false}>
        {children}
      </Content>
    ),
  });

  const renderTabName = useCallback(
    (props: Tabs.NameProps): ReactElement | null =>
      tabName == null ? null : (
        <TabScope.Provider value={props.tabKey}>{tabName(props)}</TabScope.Provider>
      ),
    [tabName],
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
    <Scope.Provider value={key}>
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
        tabName={tabName != null ? renderTabName : undefined}
      >
        {renderProp}
      </Base.Mosaic>
    </Scope.Provider>
  );
};
