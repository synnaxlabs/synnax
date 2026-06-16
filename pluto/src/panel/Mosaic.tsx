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
import { type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { Key } from "@/key";
import { Mosaic as Base } from "@/mosaic";
import {
  useDispatch,
  useEnsureRetrieved,
  useSelectRoot,
  useSelectTab,
} from "@/panel/queries";
import { Portal } from "@/portal";
import { type Tabs } from "@/tabs";

export interface MosaicTabRenderProps {
  tabKey: string;
  visible: boolean;
}

export interface MosaicTabNameProps extends Tabs.NameProps {}

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
  panelKey: panel.Key;
  selected?: string[];
  onSelect?: (tabKey: string) => void;
  children: Component.RenderProp<MosaicTabRenderProps>;
  tabName?: Component.RenderProp<MosaicTabNameProps>;
  onCreate?: (node: number, location: location.Location) => panel.Tab[];
}

const adaptToMosaic = (root: panel.Node, selected: string[] | undefined): Base.Node => {
  const preference = selected ?? [];
  const visit = (node: panel.Node | undefined, key: number): Base.Node => {
    if (node == null) return { key };
    if (node.variant === "split")
      return {
        key,
        direction: node.direction,
        size: node.size,
        first: visit(node.first, panel.childPath(key, "first")),
        last: visit(node.last, panel.childPath(key, "last")),
      };
    const tabs: Tabs.Tab[] = node.tabs.map((t) => ({
      tabKey: t.key,
      name: "",
      closable: true,
      editable: true,
    }));
    const selected =
      preference.find((key) => tabs.some((t) => t.tabKey === key)) ?? tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return visit(root, panel.ROOT_PATH);
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

export const Mosaic = ({
  panelKey: key,
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
    (node: number, location: location.Location, tabKeys?: string[]) => {
      // let tabs: panel.Tab[];
      // if (tabKeys == null) tabs = [defaultTab()];
      // else
      //   tabs = tabKeys.flatMap((raw) => {
      //     const parsed = ontology.idZ.safeParse(raw);
      //     return parsed.success ? [tabFromResource(parsed.data)] : [];
      //   });
      // if (tabs.length === 0) return;
      // const restLeaf =
      //   location === "center" ? node : panel.childPath(node, panel.splitSide(location));
      // const actions = tabs.map((tab, i) => {
      //   let payload: panel.InsertTabPayload = { tab, targetLeaf: restLeaf };
      //   if (i === 0) payload = { tab, targetLeaf: node, location };
      //   return panel.insertTab(payload);
      // });
      // dispatch({ key, actions });
      // onSelect?.(tabs[tabs.length - 1].key);
    },
    [dispatch, key, onSelect, defaultTab],
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

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => {
      const node = portalRef.current.get(props.tabKey);
      if (node == null) return null;
      return <Portal.Out node={node} />;
    },
    [portalRef],
  );

  return (
    <Key.Provider value={key}>
      {portalNodes}
      <Base.Mosaic
        {...rest}
        root={root}
        onSelect={onSelect}
        onDrop={handleDrop}
        onResize={handleResize}
        onClose={handleClose}
        onCreate={handleCreate}
        tabName={tabName}
      >
        {renderProp}
      </Base.Mosaic>
    </Key.Provider>
  );
};
