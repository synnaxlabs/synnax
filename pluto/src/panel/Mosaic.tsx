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
import { useEnsureRetrieved, useSelectRoot, useSingleDispatch } from "@/panel/queries";
import { Portal } from "@/portal";
import { type Tabs } from "@/tabs";

import { TabKeyContext } from "./Context";

export interface MosaicTabRenderProps {
  visible?: boolean;
}

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
> {
  panelKey: panel.Key;
  selected?: string[];
  children: Component.RenderProp<MosaicTabRenderProps>;
  tabName?: Component.RenderProp<Tabs.NameProps>;
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

export const Mosaic = ({
  panelKey: key,
  selected,
  children,
  onSelect,
  ...rest
}: MosaicProps): ReactElement | null => {
  useEnsureRetrieved({ key });
  const dispatch = useSingleDispatch();
  const treeRoot = useSelectRoot({});
  const root = useMemo(() => adaptToMosaic(treeRoot, selected), [treeRoot, selected]);

  const handleDrop = useCallback(
    (targetLeaf: number, key: string, location: location.Location, index?: number) =>
      dispatch(panel.moveTab({ key, targetLeaf, index, location })),
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
    [dispatch, key, onSelect],
  );

  const [portalRef, portalNodes] = Base.usePortal({
    root,
    onSelect,
    children,
  });

  const renderProp = useCallback<Tabs.RenderProp>(
    ({ tabKey }) => {
      const node = portalRef.current.get(tabKey);
      if (node == null) return null;
      return (
        <TabKeyContext value={tabKey}>
          <Portal.Out node={node} />)
        </TabKeyContext>
      );
    },
    [portalRef],
  );

  return (
    <Key.Provider value={key}>
      {portalNodes}
      <Base.Mosaic
        {...rest}
        root={root}
        onDrop={handleDrop}
        onResize={handleResize}
        onSelect={onSelect}
        onClose={handleClose}
        onCreate={handleCreate}
      >
        {renderProp}
      </Base.Mosaic>
    </Key.Provider>
  );
};
