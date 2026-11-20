// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { createPhases } from "./common/coldflow";
import { isolateTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const PLOT = "Coldflow 11 Pressures";
const LINE = "press_pt_1";
const NAMED = "Pressurant";
const TEST = "Coldflow 11";

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
 * Coldflow 11 with its phases, and the color picker of a line open on the Library tab.
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
    await createPhases(client, await client.ranges.retrieve(TEST));

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
