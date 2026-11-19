// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, TimeRange, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const PLOT = "Coldflow 11 Pressures";
const LINE = "press_pt_1";
const NAMED = "Pressurant";
const RUN = "Coldflow 11";

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

/** createStages saves the stages of the run as its child ranges, which plots draw. */
const createStages = async (client: Synnax): Promise<void> => {
  const run = await client.ranges.retrieve(RUN);
  const tr = run.timeRange;
  const [closed] = await changes(client, tr, "ox_vent");
  const [opened, shut] = await changes(client, tr, "ox_mpv");
  const stage = (name: string, color: string, start: TimeStamp, end: TimeStamp) => ({
    name,
    color,
    timeRange: new TimeRange(start, end),
    parent: run.key,
  });
  await client.ranges.create([
    stage("Pressurization", "#3774D0", closed, opened),
    stage("Flow", "#F5C400", opened, shut),
    stage("Safing", "#10B061", shut, tr.end),
  ]);
};

/** The stand's own palette: one color for each system, with a value for each theme. */
const PALETTE: ReadonlyArray<readonly [name: string, light: string, dark: string]> = [
  ["Oxidizer", "#2F6FDB", "#5B9BFF"],
  ["Fuel", "#D9363E", "#FF6369"],
  [NAMED, "#E39A12", "#FFB740"],
  ["Purge", "#2E9E63", "#3DD68C"],
  ["Vent", "#7E48C2", "#B07FF0"],
];

/**
 * Still for the named colors section of the v0.62 release notes: the pressure plot of
 * Coldflow 11 with its stages, and the color picker of a line open on the Library tab.
 * The line follows the Pressurant color of the stand's own palette, next to the
 * built-in series colors.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const palette = await client.groups.create({ name: "TPC", scope: "color" });
    await client.colors.create(
      PALETTE.map(([name, light, dark], i) => ({
        name,
        light,
        dark,
        group: palette.key,
        position: `a${i}`,
      })),
    );
    await createStages(client);

    await capture.login(session, world.project);
    await session.click(page.getByText("Review", { exact: true }).first());
    const plot = page
      .locator(".pluto-mosaic__leaf:visible")
      .filter({ hasText: PLOT })
      .first();
    await session.waitFor(plot.locator(".pluto-line-plot").first());
    await isolateTab(session, PLOT);
    await session.click(plot.getByText(PLOT, { exact: true }).first(), { zoom: false });
    await capture.showVisualizationToolbar(session);
    const toolbar = page.locator(".console-nav__drawer.lyra--location-bottom").first();
    await session.click(toolbar.getByText("Lines", { exact: true }).first(), {
      zoom: false,
    });
    const line = toolbar
      .locator(".lyra-list__item")
      .filter({ has: page.locator(`input[value="${LINE}"]`) })
      .first();
    await session.waitFor(line);
    await session.click(line.locator(".lyra-color-swatch").first(), { zoom: false });
    const picker = page.locator(".lyra-color-picker--tabbed").first();
    // A line with a fixed color opens the picker on the Custom tab.
    await session.click(picker.getByText("Library", { exact: true }), { zoom: false });
    await session.waitFor(picker.getByLabel(NAMED, { exact: true }));
    await session.click(picker.getByLabel(NAMED, { exact: true }), { zoom: false });
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
