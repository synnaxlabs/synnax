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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Flux, Ranger } from "@synnaxlabs/pluto";
import { location, TimeStamp } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

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

const ORDERS: Record<Ranger.Stage, ranger.Order> = {
  to_do: { field: "time_range_start", descending: false },
  in_progress: { field: "time_range_start", descending: true },
  completed: { field: "time_range_start", descending: true },
};

const useStage = (where: view.View["where"], stage: Ranger.Stage, now: TimeStamp) => {
  const staged = Ranger.stageWhere(stage, now);
  return Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: {
      where: where == null ? staged : { and: [where as ranger.Where, staged] },
      order: ORDERS[stage],
    },
  });
};

const Internal = () => {
  const [where, setWhere] = useState(View.useContext().getInitialView().where);
  const [now, setNow] = useState(() => TimeStamp.now());
  const toDo = useStage(where, "to_do", now);
  const inProgress = useStage(where, "in_progress", now);
  const completed = useStage(where, "completed", now);
  const stages = [toDo, inProgress, completed];
  // A range that crossed into another stage since now, such as one created to start
  // at once, lists again under a later now.
  useEffect(() => {
    const at = TimeStamp.now();
    const crossed = stages.some(({ data, getItem }, i) =>
      getItem(data).some(
        ({ timeRange }) =>
          Ranger.getStage(timeRange, now) === Ranger.STAGES[i] &&
          Ranger.getStage(timeRange, at) !== Ranger.STAGES[i],
      ),
    );
    if (crossed) setNow(at);
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
  const handleSearch = (term: string) => stages.forEach((s) => s.search(term));
  const openCreate = Range.useCreateModal();
  const handleCreate = () => openCreate();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <View.Form
      data={shown.flatMap((s) => s.data)}
      getItem={toDo.getItem}
      subscribe={toDo.subscribe}
      answered={stages.every((s) => s.answered)}
      more={handleMore}
      search={handleSearch}
      onWhereChange={setWhere}
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
