// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/explorer/Explorer.css";

import { ranger, type view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Store } from "@synnaxlabs/lyra/store";
import { Access, Flux, Ranger } from "@synnaxlabs/pluto";
import { location, TimeRange, TimeStamp, unique } from "@synnaxlabs/x";
import { useState } from "react";

import { ContextMenu } from "@/feature/range/list/ContextMenu";
import { Item, type ItemProps } from "@/feature/range/list/Item";
import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { View } from "@/platform/view";

const item = Component.renderProp((props: ItemProps) => (
  <Item {...props} role="listitem" />
));
const contextMenu = Component.renderProp(ContextMenu);

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="range" icon="Range">
    <Internal />
  </View.Frame>
);

const ORDERS: Record<ranger.Stage, ranger.Order> = {
  to_do: { field: "time_range_start", descending: false },
  in_progress: { field: "time_range_start", descending: true },
  completed: { field: "time_range_start", descending: true },
};

const PAGE_SIZE = 50;
const INITIAL_SIZES = ranger.STAGES.map(() => PAGE_SIZE);

interface Listing {
  /** The time the stages list their ranges at. */
  at: TimeStamp;
  /** How many ranges each stage reads at first. */
  sizes: number[];
}

const useStage = (where: view.View["where"], stage: ranger.Stage, listing: Listing) => {
  const staged = Ranger.stageWhere(stage, listing.at);
  const list = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: {
      where: where == null ? staged : { and: [where as ranger.Where, staged] },
      order: ORDERS[stage],
    },
    pageSize: PAGE_SIZE,
    initialSize: listing.sizes[ranger.STAGES.indexOf(stage)],
  });
  // The explorer reads the ranges in render, so it subscribes to them.
  const ranges = Store.useItems({
    keys: list.data,
    subscribe: list.subscribe,
    getItem: list.getItem,
  });
  return { ...list, ranges };
};

type Stage = ReturnType<typeof useStage>;

// The instant a range leaves the stage it lists under.
const leaves = (stage: ranger.Stage, range: ranger.Range): TimeStamp | null => {
  const { start, end } = new TimeRange(range.timeRange).makeValid();
  if (stage === "to_do") return start;
  if (stage === "in_progress") return end;
  return null;
};

const nextCrossing = (stages: Stage[], now: TimeStamp): TimeStamp | null => {
  let next: TimeStamp | null = null;
  stages.forEach(({ ranges }, i) =>
    ranges.forEach((range) => {
      const instant = leaves(ranger.STAGES[i], range);
      if (instant?.after(now) === true && (next == null || instant.before(next)))
        next = instant;
    }),
  );
  return next;
};

const Internal = () => {
  const [where, setWhere] = useState(View.useContext().getInitialView().where);
  const [listing, setListing] = useState<Listing>(() => ({
    at: TimeStamp.now(),
    sizes: INITIAL_SIZES,
  }));
  // A new filter or search reads its first page at the usual size.
  const resetSizes = () => setListing((prev) => ({ ...prev, sizes: INITIAL_SIZES }));
  const toDo = useStage(where, "to_do", listing);
  const inProgress = useStage(where, "in_progress", listing);
  const completed = useStage(where, "completed", listing);
  const stages = [toDo, inProgress, completed];
  const now = useNow((at) => nextCrossing(stages, at));
  // A range that left its stage, such as one created to start at once, lists again
  // under the new time. Each stage reads as many ranges as it holds, so the list keeps
  // its place.
  const crossed = stages.some(({ ranges }, i) =>
    ranges.some(
      ({ timeRange }) =>
        Ranger.getStage(timeRange, listing.at) === ranger.STAGES[i] &&
        Ranger.getStage(timeRange, now) !== ranger.STAGES[i],
    ),
  );
  if (crossed)
    setListing({
      at: now,
      sizes: stages.map(({ data }) => Math.max(PAGE_SIZE, data.length)),
    });
  // A stage shows once every stage before it is complete, so pages arrive in order.
  const incomplete = stages.findIndex((s) => !s.complete);
  const shown = incomplete === -1 ? stages : stages.slice(0, incomplete + 1);
  const handleMore = () => {
    stages.forEach((s) => {
      if (!s.answered) s.more();
    });
    const first = stages[incomplete];
    if (first?.answered) first.more();
  };
  const handleSearch = (term: string) => {
    resetSizes();
    stages.forEach((s) => s.search(term));
  };
  const handleWhereChange = (next: view.View["where"]) => {
    resetSizes();
    setWhere(next);
  };
  const openCreate = Range.useCreateModal();
  const handleCreate = () => openCreate();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <View.Form
      // A stage shows its last ranges until its new ones land, so a range that crossed
      // can show in two stages for a moment.
      data={unique.unique(shown.flatMap((s) => s.data))}
      getItem={toDo.getItem}
      subscribe={toDo.subscribe}
      answered={stages.every((s) => s.answered)}
      more={handleMore}
      search={handleSearch}
      onWhereChange={handleWhereChange}
      itemHeight={40}
    >
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem />
        </View.FilterMenu>
        <View.Search />
        <Label.Filter.Chips />
        {hasCreatePermission && (
          <Button.Button
            onClick={handleCreate}
            tooltipLocation={location.BOTTOM_LEFT}
            tooltip="Create range"
            variant="filled"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items
        className={CSS.BE("range", "explorer-list")}
        role="list"
        aria-label="Ranges"
        contextMenu={contextMenu}
      >
        {item}
      </View.Items>
    </View.Form>
  );
};
