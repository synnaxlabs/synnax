// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import {
  Flex,
  Mosaic as Base,
  Panel as PlutoPanel,
  type Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

export interface MosaicProps {
  panelKey: panel.Key;
  windowKey: string;
}

// adaptToMosaic walks the typed panel tree and emits the pluto Base.Mosaic.Node
// shape it expects, assigning path-derived keys (root=1, 2k=first, 2k+1=last).
// Display info for each tab is derived from the referenced ontology resource
// at render time; for Phase 1 we render the resource key as a placeholder
// name until the per-viz renderer bridge lands.
const adaptToMosaic = (
  root: panel.Node | null | undefined,
  activeTab: string | null,
): Base.Node => {
  const visit = (node: panel.Node | null | undefined, key: number): Base.Node => {
    if (node == null) return { key };
    if (node.split != null) {
      const dir: "x" | "y" = node.split.direction;
      return {
        key,
        direction: dir === "x" ? "x" : "y",
        size: node.split.size,
        first: visit(node.split.first, key * 2),
        last: visit(node.split.last, key * 2 + 1),
      };
    }
    if (node.leaf == null) return { key, tabs: [] };
    const tabs: Tabs.Tab[] = node.leaf.tabs.map((t) => ({
      tabKey: t.key,
      name: `${t.resource.type}:${t.resource.key.slice(0, 8)}`,
      closable: true,
    }));
    const selected =
      activeTab != null && tabs.some((t) => t.tabKey === activeTab)
        ? activeTab
        : tabs[0]?.tabKey;
    return { key, tabs, selected };
  };
  return visit(root, 1);
};

const PlaceholderContent = ({ tabKey }: Tabs.Tab): ReactElement => (
  <Flex.Box grow center>
    <Text.Text level="h3" color={9}>
      Tab {tabKey}
    </Text.Text>
    <Text.Text level="small" color={7}>
      Viz rendering bridge lands in a follow-up.
    </Text.Text>
  </Flex.Box>
);

// Mosaic renders the active panel's typed mosaic tree via pluto's Base.Mosaic.
// Tab content is currently a placeholder; gestures (drop, resize, split) wire
// to panel.useDispatch so other consoles see the same mutations through the
// action channel.
export const Mosaic = ({ panelKey, windowKey }: MosaicProps): ReactElement | null => {
  const dispatch = useDispatch();
  const activeTab = Layout.useSelectActiveTabKey();
  const { data: p } = PlutoPanel.useRetrieve({ key: panelKey });
  const { dispatch: dispatchAction } = PlutoPanel.useDispatch();

  const root = useMemo(() => adaptToMosaic(p?.root, activeTab), [p?.root, activeTab]);

  const handleSelect = useCallback(
    (tabKey: string) => {
      dispatch(Layout.setActiveTab({ windowKey, key: tabKey }));
    },
    [dispatch, windowKey],
  );

  const handleDrop = useCallback(
    (key: number, tabKey: string, _loc: location.Location, index?: number) => {
      // For Phase 1 same-panel drops translate to MoveTab. Cross-panel drag
      // arrives via Haul events handled at a higher layer.
      dispatchAction({
        key: panelKey,
        actions: [
          {
            type: "move_tab",
            moveTab: { key: tabKey, targetLeaf: key, index: index ?? 0 },
          },
        ],
      });
    },
    [dispatchAction, panelKey],
  );

  const handleResize = useCallback(
    (key: number, size: number) => {
      dispatchAction({
        key: panelKey,
        actions: [{ type: "resize_split", resizeSplit: { split: key, size } }],
      });
    },
    [dispatchAction, panelKey],
  );

  const handleClose = useCallback(
    (tabKey: string) => {
      dispatchAction({
        key: panelKey,
        actions: [{ type: "remove_tab", removeTab: { key: tabKey } }],
      });
    },
    [dispatchAction, panelKey],
  );

  const handleCreate = useCallback(
    (key: number, loc: location.Location, _tabKeys?: string[]) => {
      // Splits the target leaf into a new empty sibling. Content insertion
      // comes from a separate gesture (drag from sidebar, palette command).
      dispatchAction({
        key: panelKey,
        actions: [
          {
            type: "split_leaf",
            splitLeaf: {
              leaf: key,
              location: loc,
              size: 0.5,
            },
          },
        ],
      });
    },
    [dispatchAction, panelKey],
  );

  if (p == null) return null;

  return (
    <Base.Mosaic
      rounded={1}
      bordered
      borderColor={5}
      background={0}
      root={root}
      activeTab={activeTab ?? undefined}
      onSelect={handleSelect}
      onDrop={handleDrop}
      onResize={handleResize}
      onClose={handleClose}
      onCreate={handleCreate}
    >
      {(tab) => <PlaceholderContent {...tab} />}
    </Base.Mosaic>
  );
};
