// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/tree/Layout.css";

import { type ranger, type view } from "@synnaxlabs/client";
import { type Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement, type ReactNode } from "react";

import { ContextMenu } from "@/feature/range/tree/ContextMenu";
import { FlatItem, HighlightContext, Item } from "@/feature/range/tree/Item";
import { Timeline } from "@/feature/range/tree/Timeline";
import { type Node, NodeProvider } from "@/feature/range/tree/tree";
import { CSS } from "@/platform/css";

export const ORDERS: Record<view.RangeSort, ranger.Order> = {
  newest: { field: "time_range_start", descending: true },
  oldest: { field: "time_range_start", descending: false },
  name: { field: "name", descending: false },
};

/** @returns the height in pixels of each range in the layout. */
export const itemHeight = (layout: view.RangeLayout): number =>
  layout === "timeline" ? 33 : 40;

const contextMenu = Component.renderProp(ContextMenu);
const item = Component.renderProp(Item);
const flatItem = Component.renderProp(FlatItem);

interface LayoutInfo {
  name: string;
  icon: Icon.ReactElement;
}

const LAYOUTS: Record<view.RangeLayout, LayoutInfo> = {
  tree: { name: "Tree", icon: <Icon.TreeView /> },
  list: { name: "List", icon: <Icon.Menu /> },
  timeline: { name: "Timeline", icon: <Icon.Timeline /> },
};

export interface LayoutSelectProps {
  value: view.RangeLayout;
  onChange: (value: view.RangeLayout) => void;
  /** The layouts to offer, in order. */
  layouts: view.RangeLayout[];
  size?: Button.ButtonProps["size"];
}

export const LayoutSelect = ({
  value,
  onChange,
  layouts,
  size = "small",
}: LayoutSelectProps): ReactElement => (
  <Select.Buttons<view.RangeLayout> size={size} value={value} onChange={onChange}>
    {layouts.map((layout) => (
      <Select.Item<view.RangeLayout> key={layout} itemKey={layout} size={size}>
        {LAYOUTS[layout].icon}
        {LAYOUTS[layout].name}
      </Select.Item>
    ))}
  </Select.Buttons>
);

export interface LayoutProps {
  layout: view.RangeLayout;
  nodes: Node[];
  expanded: ranger.Key[];
  onExpand: (key: ranger.Key, expanded: boolean) => void;
  /** Reads the next page of the children of a range. */
  more: (key: ranger.Key) => void;
  bounds?: ranger.Range["timeRange"];
  emptyContent?: ReactNode;
  /** The search term to tint in each name. */
  highlight?: string;
  /** Shows after the last range. */
  footer?: ReactNode;
}

/**
 * The ranges of a tree, as rows or as lanes over a time axis. The list layout shows
 * each range on its own, with no children under it. It reads the ranges from the
 * enclosing list frame.
 */
export const Layout = ({
  layout,
  nodes,
  expanded,
  onExpand,
  more,
  bounds,
  emptyContent,
  highlight,
  footer,
}: LayoutProps): ReactElement => {
  const menuProps = Menu.useContextMenu();
  const content =
    layout === "timeline" ? (
      <Timeline
        nodes={nodes}
        bounds={bounds}
        emptyContent={emptyContent}
        footer={footer}
        onContextMenu={menuProps.open}
      />
    ) : (
      <List.Scroll
        grow
        className={CSS.BE("range-tree", "list")}
        onContextMenu={menuProps.open}
        role="list"
        aria-label="Ranges"
      >
        <List.Items<ranger.Key> emptyContent={emptyContent}>
          {layout === "list" ? flatItem : item}
        </List.Items>
        {footer}
      </List.Scroll>
    );
  return (
    <NodeProvider nodes={nodes} expanded={expanded} onExpand={onExpand} more={more}>
      <HighlightContext value={highlight}>
        <Menu.ContextMenu menu={contextMenu} {...menuProps}>
          {content}
        </Menu.ContextMenu>
      </HighlightContext>
    </NodeProvider>
  );
};
