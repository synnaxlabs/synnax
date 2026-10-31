// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  panel,
  type ranger,
  type Synnax,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

export const viewport = { width: 1280, height: 800 };

// A range lists its labels in key order, so fixed keys hold the order on both themes.
const LABELS = {
  LOX: { key: "00000001-0000-4000-8000-000000000000", color: "#3B7DDB" },
  Nominal: { key: "00000002-0000-4000-8000-000000000000", color: "#10B061" },
  Anomaly: { key: "00000003-0000-4000-8000-000000000000", color: "#DC1360" },
};

type Label = keyof typeof LABELS;

const CAMPAIGN = "LOX coldflow campaign";
const LEAK_CHECK = "OX leak check";
const REVIEWED = "Coldflow 11";

/** The stand's earlier runs in the order the toolbar lists them, with their labels. */
const RUNS: ReadonlyArray<readonly [name: string, labels: Label[]]> = [
  ["Coldflow 9", ["LOX", "Nominal"]],
  ["Coldflow 10", ["LOX", "Anomaly"]],
  [REVIEWED, ["LOX", "Nominal"]],
];

/** changes returns the times at which a valve on the stand changes state during tr. */
const changes = async (
  client: Synnax,
  tr: TimeRange,
  valve: string,
): Promise<TimeStamp[]> => {
  const times = Array.from(await client.read(tr, "daq_time"));
  const states = Array.from(await client.read(tr, `${valve}_state`));
  return states.flatMap((s, i) =>
    i > 0 && s !== states[i - 1] ? [new TimeStamp(times[i] as bigint)] : [],
  );
};

/**
 * Still for the hero of docs page `reference/ranges`: the Ranges toolbar lists the TPC
 * stand's coldflow campaign and its tests with their labels, next to the Review panel,
 * which plots Coldflow 11 with its stages drawn as child ranges.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await client.labels.create(
      Object.entries(LABELS).map(([name, l]) => ({ name, ...l })),
    );
    const label = async (range: ranger.Range, labels: Label[]): Promise<void> =>
      await client.labels.label(
        range.ontologyID,
        labels.map((l) => LABELS[l].key),
      );
    const runs = await Promise.all(
      RUNS.map(async ([name, labels]) => {
        const range = await client.ranges.retrieve(name);
        await label(range, labels);
        return range;
      }),
    );
    const first = runs[0].timeRange.start;
    // The campaign has no end, so it stays in progress.
    const campaign = await client.ranges.create({
      name: CAMPAIGN,
      timeRange: new TimeRange(first.sub(TimeSpan.minutes(12)), TimeStamp.MAX),
    });
    await label(campaign, ["LOX"]);
    const leakCheck = await client.ranges.create({
      name: LEAK_CHECK,
      timeRange: new TimeRange(
        first.sub(TimeSpan.minutes(9)),
        first.sub(TimeSpan.minutes(3)),
      ),
    });
    await label(leakCheck, ["LOX", "Nominal"]);

    // The stages of the reviewed run, from the valve states the stand recorded.
    const run = runs[runs.length - 1];
    const tr = run.timeRange;
    const [closed] = await changes(client, tr, "ox_vent");
    const [opened, shut] = await changes(client, tr, "ox_mpv");
    const stage = (name: string, color: string, start: TimeStamp, end: TimeStamp) => ({
      name,
      color,
      timeRange: new TimeRange(start, end),
      parent: { key: run.key },
    });
    await client.ranges.create([
      stage("Pressurization", "#3774D0", closed, opened),
      stage("Flow", "#F5C400", opened, shut),
      stage("Safing", "#10B061", shut, tr.end),
    ]);

    // The hero crops the window just below the tab of the temperature plot.
    const review = (await client.panels.retrieve({ searchTerm: "Review" })).find(
      (p) => p.name === "Review",
    );
    if (review == null) throw new Error("the TPC world has no Review panel");
    const resized = await client.panels.dispatch(
      review.key,
      panel.resizeSplit({ split: 2, size: 0.687 }),
    );
    if (!resized) throw new Error("the Review panel is not cached");

    await capture.login(session, world.project);
    await session.click(page.getByText("Review", { exact: true }).first());
    await session.waitFor(page.locator(".pluto-line-plot:visible").first());
    await session.settle(1500);
    // Favorites join the toolbar in order, and the last one becomes the active range.
    for (const name of [CAMPAIGN, LEAK_CHECK, ...RUNS.map(([name]) => name)]) {
      const item = page
        .locator(`.console-range__list-item[aria-label="${name}"]`)
        .first();
      await session.waitFor(item);
      await capture.contextMenu(
        session,
        item.getByText(name, { exact: true }).first(),
        "Favorite",
      );
      session.endZoom();
      await session.settle(300);
    }
    await capture.openToolbar(session, "range");
    const toolbar = page.locator(".console-nav__drawer.lyra--location-left").first();
    await session.waitFor(toolbar.getByText(REVIEWED, { exact: true }).first());
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // The plots redraw at their new size on wall time, which virtual ticks outrun.
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
