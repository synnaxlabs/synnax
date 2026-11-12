// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type view } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Flux } from "@synnaxlabs/pluto";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import {
  itemHeight,
  LayoutSelect,
  ORDERS,
  TreeLayout,
} from "@/feature/range/next/Explorer";
import { AddButton, Section } from "@/feature/range/next/overview/Section";
import { childrenQuery, useExpanded, useTree } from "@/feature/range/next/tree";
import { CSS } from "@/platform/css";

/**
 * A range longer than a day, such as a campaign, holds many children, so it lists them
 * newest first. A shorter one shows its children, such as phases, on a timeline.
 */
const defaultLayout = (range: ranger.Range, now: TimeStamp): view.RangeLayout => {
  const { start, end } = range.timeRange.makeValid();
  const span = start.span(end.before(now) ? end : now);
  return span.greaterThan(TimeSpan.DAY) ? "tree" : "timeline";
};

/** A tree shows the newest child first, and a timeline shows its lanes in order. */
const childOrder = (layout: view.RangeLayout): ranger.Order =>
  layout === "tree" ? ORDERS.newest : ORDERS.oldest;

const CHILD_LAYOUTS: view.RangeLayout[] = ["tree", "timeline"];

interface ChildrenProps {
  range: ranger.Range;
  onCreate?: () => void;
}

export const Children = ({ range, onCreate }: ChildrenProps): ReactElement => {
  const openedAt = useNow(() => null);
  const [chosenLayout, setLayout] = useState<view.RangeLayout | null>(null);
  const layout = chosenLayout ?? defaultLayout(range, openedAt);
  const list = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: childrenQuery(range.key, childOrder(layout)),
    pageSize: 100,
  });
  const [expanded, toggle] = useExpanded();
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const tree = useTree({ roots: list.data, expanded });
  return (
    <Section
      title="Child ranges"
      icon={<Icon.Range />}
      className={CSS.BE("range-overview", "children")}
      actions={
        <>
          <LayoutSelect value={layout} onChange={setLayout} layouts={CHILD_LAYOUTS} />
          {onCreate != null && (
            <AddButton tooltip="Create child range" onClick={onCreate} />
          )}
        </>
      }
    >
      <Select.Frame
        multiple
        data={tree.nodes.map((n) => n.key)}
        getItem={list.getItem}
        subscribe={list.subscribe}
        value={selected}
        onChange={setSelected}
        onFetchMore={list.more}
        virtual
        itemHeight={itemHeight(layout)}
      >
        <TreeLayout
          layout={layout}
          nodes={tree.nodes}
          expanded={expanded}
          onExpand={toggle}
          more={tree.more}
          bounds={range.timeRange}
        />
      </Select.Frame>
    </Section>
  );
};
