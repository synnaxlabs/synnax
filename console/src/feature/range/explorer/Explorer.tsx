// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/explorer/Explorer.css";

import { ranger, view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, View as PView } from "@synnaxlabs/pluto";
import { binary, location } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Chips, isEmpty, MenuItems } from "@/feature/range/explorer/Filter";
import { MATCH_LIMIT, useSearchTree } from "@/feature/range/explorer/searchTree";
import { Tree } from "@/feature/range/tree";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { View } from "@/platform/view";

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
    query: { where: scoped, order: Tree.ORDERS[sort] },
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
    order: Tree.ORDERS[sort],
    getItem: root.getItem,
  });
  const [expanded, toggle] = Tree.useExpanded();
  const tree = Tree.use(
    capped
      ? {
          roots: search.roots,
          expanded: search.expanded,
          fixed: search.fixed,
          order: Tree.ORDERS[sort],
        }
      : { roots: root.data, expanded, order: Tree.ORDERS[sort] },
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
      itemHeight={Tree.itemHeight(layout)}
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
          <Tree.LayoutSelect
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
      <Tree.Layout
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
