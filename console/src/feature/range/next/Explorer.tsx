// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/next/Range.css";

import { ranger, view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, View as PView } from "@synnaxlabs/pluto";
import { binary, location } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { ContextMenu } from "@/feature/range/next/ContextMenu";
import { Chips, isEmpty, MenuItems } from "@/feature/range/next/Filter";
import { FlatItem, HighlightContext, TreeItem } from "@/feature/range/next/Item";
import { MATCH_LIMIT, useSearchTree } from "@/feature/range/next/searchTree";
import { Timeline } from "@/feature/range/next/Timeline";
import {
  type Node,
  NodeProvider,
  useExpanded,
  useTree,
} from "@/feature/range/next/tree";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { View } from "@/platform/view";

export const ORDERS: Record<view.RangeSort, ranger.Order> = {
  newest: { field: "time_range_start", descending: true },
  oldest: { field: "time_range_start", descending: false },
  name: { field: "name", descending: false },
};

/** @returns the height in pixels of each range in the layout. */
export const itemHeight = (layout: view.RangeLayout): number =>
  layout === "timeline" ? 33 : 40;

const contextMenu = Component.renderProp(ContextMenu);
const treeItem = Component.renderProp(TreeItem);
const flatItem = Component.renderProp(FlatItem);

interface SortSelectProps {
  value: view.RangeSort;
  onChange: (value: view.RangeSort) => void;
}

const SORT_ICONS: Record<view.RangeSort, Icon.ReactElement> = {
  newest: <Icon.Arrow.Down />,
  oldest: <Icon.Arrow.Up />,
  name: <Icon.Text />,
};

const SortSelect = ({ value, onChange }: SortSelectProps): ReactElement => (
  <Select.Simple<view.RangeSort>
    resourceName="order"
    searchHidden
    value={value}
    onChange={onChange}
    triggerProps={{ variant: "text", size: "medium", hideCaret: true, textColor: 9 }}
  >
    <Select.Item<view.RangeSort> itemKey="newest">
      {SORT_ICONS.newest}
      Newest first
    </Select.Item>
    <Select.Item<view.RangeSort> itemKey="oldest">
      {SORT_ICONS.oldest}
      Oldest first
    </Select.Item>
    <Select.Item<view.RangeSort> itemKey="name">
      {SORT_ICONS.name}
      Name
    </Select.Item>
  </Select.Simple>
);

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

export interface TreeLayoutProps {
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
export const TreeLayout = ({
  layout,
  nodes,
  expanded,
  onExpand,
  more,
  bounds,
  emptyContent,
  highlight,
  footer,
}: TreeLayoutProps): ReactElement => {
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
        className={CSS.BE("range-explorer", "list")}
        onContextMenu={menuProps.open}
        role="list"
        aria-label="Ranges"
      >
        <List.Items<ranger.Key> emptyContent={emptyContent}>
          {layout === "list" ? flatItem : treeItem}
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

const NoRanges = (): ReactElement => {
  const openCreate = Range.useCreateModal();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No ranges match"
      action={canCreate ? "Create range" : undefined}
      onClick={() => openCreate()}
    />
  );
};

const NO_RANGES = <NoRanges />;

const EXPLORER_LAYOUTS: view.RangeLayout[] = ["tree", "list", "timeline"];

/**
 * The sort or the layout of the open view. A stored view reads it from the view and
 * saves a change to it. A static view holds it here.
 */
const useViewField = <F extends "sort" | "layout">(
  field: F,
): [view.RangeView[F], (next: view.RangeView[F]) => void] => {
  const { selected, staticViews, editable, getInitialView } = View.useContext();
  const stored = !staticViews.includes(selected);
  const [local, setLocal] = useState(() => (getInitialView() as view.RangeView)[field]);
  const live = PView.useResult(stored ? { key: selected } : null).data;
  const { update } = PView.useSetFields();
  if (!stored) return [local, setLocal];
  const value = live?.type === "range" ? live[field] : local;
  return [
    value,
    (next) => {
      if (editable)
        update({ key: selected, actions: view.set({ path: [field], value: next }) });
    },
  ];
};

const Internal = (): ReactElement => {
  const { getInitialView, searchTerm } = View.useContext();
  const [where, setWhere] = useState(() => getInitialView().where);
  const [sort, setSort] = useViewField("sort");
  const [layout, setLayout] = useViewField("layout");
  const flat = layout === "list";
  // A search in a tree shows its first matches only, so the tree holds still.
  const searching = (searchTerm !== "" || !isEmpty(where)) && !flat;
  const scoped: ranger.Where | undefined =
    flat || searching
      ? where
      : where == null
        ? { nested: false }
        : { and: [where, { nested: false }] };
  const root = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: { where: scoped, order: ORDERS[sort] },
    initialSize: searching ? MATCH_LIMIT + 1 : undefined,
    // A search in a tree reads its first page only.
    eager: searching,
  });
  // The list shows its last answer until a new search lands, so the tree follows the
  // term that the shown ranges answer.
  const capped = (root.searchTerm !== "" || !isEmpty(where)) && !flat;
  const search = useSearchTree({
    matches: capped ? root.getItem(root.data.slice(0, MATCH_LIMIT)) : null,
    scope: `${root.searchTerm}\n${binary.JSON_CODEC.encodeString(where ?? null)}`,
    order: ORDERS[sort],
    getItem: root.getItem,
  });
  const [expanded, toggle] = useExpanded();
  const tree = useTree(
    capped
      ? {
          roots: search.roots,
          expanded: search.expanded,
          fixed: search.fixed,
          order: ORDERS[sort],
        }
      : { roots: root.data, expanded, order: ORDERS[sort] },
  );
  const openCreate = Range.useCreateModal();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const answered = capped ? search.answered : root.answered;
  let emptyContent: ReactElement | null = answered ? NO_RANGES : null;
  if (capped && search.error != null)
    emptyContent = (
      <Status.Summary
        className={CSS.BE("range-explorer", "message")}
        variant="error"
        message="Failed to search ranges"
        description={search.error}
      />
    );
  const footer = capped && root.data.length > MATCH_LIMIT && (
    <Text.Text level="small" color={9} className={CSS.BE("range-explorer", "more")}>
      Showing the first {MATCH_LIMIT} matches.{" "}
      {root.searchTerm === ""
        ? "Add a filter to narrow them."
        : "Type more to narrow the search."}
    </Text.Text>
  );
  return (
    <View.Form
      data={tree.nodes.map((n) => n.key)}
      getItem={root.getItem}
      subscribe={root.subscribe}
      answered={answered}
      more={root.more}
      search={root.search}
      onWhereChange={setWhere}
      itemHeight={itemHeight(layout)}
    >
      <View.Toolbar>
        <Flex.Box x align="center" gap="small" grow wrap>
          <View.FilterMenu>
            <MenuItems />
          </View.FilterMenu>
          <Chips />
        </Flex.Box>
        <Flex.Box x align="center" gap="small">
          {searchTerm === "" && <SortSelect value={sort} onChange={setSort} />}
          <LayoutSelect
            value={layout}
            onChange={setLayout}
            layouts={EXPLORER_LAYOUTS}
            size="medium"
          />
          {canCreate && (
            <Button.Button
              variant="outlined"
              size="medium"
              tooltip="Create range"
              tooltipLocation={location.BOTTOM_LEFT}
              onClick={() => openCreate()}
            >
              <Icon.Add />
            </Button.Button>
          )}
        </Flex.Box>
      </View.Toolbar>
      <TreeLayout
        layout={layout}
        nodes={tree.nodes}
        expanded={capped ? search.expanded : expanded}
        onExpand={capped ? search.onExpand : toggle}
        more={tree.more}
        emptyContent={emptyContent}
        highlight={root.searchTerm === "" ? undefined : root.searchTerm}
        footer={footer}
      />
    </View.Form>
  );
};

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="range" icon="Range" alwaysEditable searchInViews>
    <Internal />
  </View.Frame>
);
