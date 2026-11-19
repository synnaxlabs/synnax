// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Store } from "@synnaxlabs/lyra/store";
import { Flux, Ranger } from "@synnaxlabs/pluto";
import { array, TimeStamp, unique } from "@synnaxlabs/x";
import { useState } from "react";

import { Session } from "@/session";

const NEWEST: ranger.Order = { field: "time_range_start", descending: true };
// Recent shows how long ago each range ended, so it sorts by the end.
const LATEST_END: ranger.Order = { field: "time_range_end", descending: true };
const RUNNING_PAGE = 100;
const STAGES: ranger.Where = { stages: ["in_progress", "completed"] };

const latestEndFirst = (a: ranger.Range, b: ranger.Range): number =>
  TimeStamp.sort(b.timeRange.end, a.timeRange.end);

export interface Forest {
  roots: ranger.Range[];
  children: Map<ranger.Key, ranger.Range[]>;
}

/** Nests each range under its parent when the parent is in the same set. */
export const forest = (ranges: ranger.Range[]): Forest => {
  const keys = new Set(ranges.map((r) => r.key));
  const roots: ranger.Range[] = [];
  const children = new Map<ranger.Key, ranger.Range[]>();
  for (const r of ranges)
    if (r.parent == null || !keys.has(r.parent)) roots.push(r);
    else {
      const siblings = children.get(r.parent) ?? [];
      siblings.push(r);
      children.set(r.parent, siblings);
    }
  return { roots, children };
};

export interface Groups {
  running: ranger.Range[];
  recent: ranger.Range[];
  favorites: ranger.Range[];
  /** Whether running holds every running range. */
  runningComplete: boolean;
  /** Reads the next page of running ranges. */
  moreRunning: () => void;
  /** Whether a list reads. The groups keep the last answers until it lands. */
  reading: boolean;
  getItem: ReturnType<typeof Flux.useList<ranger.Key, ranger.Range>>["getItem"];
  subscribe: ReturnType<typeof Flux.useList<ranger.Key, ranger.Range>>["subscribe"];
}

export const useGroups = (): Groups => {
  // The Core answers the stage filters as of `at`, so `at` moves to each start or end.
  const [at, setAt] = useState(() => TimeStamp.now());
  // Show more grows the size, so a read for a new `at` keeps the pages already shown.
  const [runningSize, setRunningSize] = useState(RUNNING_PAGE);
  const runningList = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: Ranger.stageWhere("in_progress", at), order: NEWEST },
    initialSize: runningSize,
    eager: true,
  });
  const recentList = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: Ranger.stageWhere("completed", at), order: LATEST_END },
    initialSize: 250,
    eager: true,
  });
  // Each list shows its last answer until its answer for a new `at` lands, and the two
  // land apart. Until both land, the groups keep the ranges of the last answers, so a
  // range that ends moves to Recent at once and never shows twice or drops out.
  const [held, setHeld] = useState<ranger.Key[]>([]);
  const reading = runningList.variant === "loading" || recentList.variant === "loading";
  const keys = unique.unique([
    ...runningList.data,
    ...recentList.data,
    ...(reading ? held : []),
  ]);
  if (!reading && !array.equal(held, keys)) setHeld(keys);
  const ranges = Store.useItems({
    keys,
    subscribe: runningList.subscribe,
    getItem: runningList.getItem,
  });
  const running = ranges.filter(
    (r) => Ranger.getStage(r.timeRange, at) === "in_progress",
  );
  const recent = ranges
    .filter((r) => Ranger.getStage(r.timeRange, at) === "completed")
    .sort(latestEndFirst);
  const upcoming = Ranger.useNextRanges(STAGES, at);
  const crossing = Ranger.nextCrossing(STAGES, upcoming, at);
  const now = useNow(() => crossing);
  if (crossing != null && !now.before(crossing)) setAt(now);
  const favoriteKeys = Session.Range.useSelectFavoriteKeys();
  const { data: favoriteData } = Ranger.useResultMultiple(
    favoriteKeys.length === 0 ? null : { keys: favoriteKeys },
  );
  const byKey = new Map((favoriteData ?? []).map((r) => [r.key, r]));
  const favorites = favoriteKeys
    .map((k) => byKey.get(k))
    .filter((r): r is ranger.Range => r != null);
  return {
    running,
    recent,
    favorites,
    runningComplete: runningList.complete,
    moreRunning: () => setRunningSize((size) => size + RUNNING_PAGE),
    reading,
    getItem: runningList.getItem,
    subscribe: runningList.subscribe,
  };
};
