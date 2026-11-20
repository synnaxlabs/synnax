// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, type ranger, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { createPhases } from "./common/coldflow";

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

/** The stand's earlier tests in the order the toolbar lists them, with their labels. */
const TESTS: ReadonlyArray<readonly [name: string, labels: Label[]]> = [
  ["Coldflow 9", ["LOX", "Nominal"]],
  ["Coldflow 10", ["LOX", "Anomaly"]],
  [REVIEWED, ["LOX", "Nominal"]],
];

/**
 * Still for the hero of docs page `reference/ranges`: the Ranges toolbar lists the TPC
 * stand's running coldflow campaign, its starred ranges, and its recent tests, next to
 * the Review panel, which plots Coldflow 11 with its phases drawn as child ranges.
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
        range.resourceID,
        labels.map((l) => LABELS[l].key),
      );
    const tests = await Promise.all(
      TESTS.map(async ([name, labels]) => {
        const range = await client.ranges.retrieve(name);
        await label(range, labels);
        return range;
      }),
    );
    const first = tests[0].timeRange.start;
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

    await createPhases(client, tests[tests.length - 1]);

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
    // Recent lists every test, so only the ranges under review get a star.
    for (const name of [CAMPAIGN, REVIEWED]) {
      const item = page.getByRole("listitem", { name, exact: true }).first();
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
