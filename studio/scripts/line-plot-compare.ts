// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { openProject } from "./common/desktop";

export const desktop = true;

const PROJECT = "My project";
const CHANNEL = "chamber_pressure";
const PLOT = "Hotfire review";
const PERIOD = TimeSpan.milliseconds(40);
const SAMPLES = 1000;

// Each test fires a day after the one before it, with its own peak and ignition delay.
const TESTS = [
  { name: "Hotfire 07", daysAgo: 3, peak: 250, delay: 0 },
  { name: "Hotfire 08", daysAgo: 2, peak: 236, delay: 0.6 },
  { name: "Hotfire 09", daysAgo: 1, peak: 266, delay: 1.2 },
];

// Returns the chamber pressure in psi at t seconds into a test.
const pressure = (t: number, peak: number, delay: number): number => {
  const ambient = 14.7;
  const ignition = 6 + delay;
  const shutdown = 30 + delay;
  if (t < ignition) return ambient + 0.3 * Math.sin(t * 7);
  if (t < ignition + 2) return ambient + ((peak - ambient) * (t - ignition)) / 2;
  if (t < shutdown) return peak + 4 * Math.sin(t * 3) + 2 * Math.sin(t * 11);
  if (t < shutdown + 2) return peak - ((peak - ambient) * (t - shutdown)) / 2;
  return ambient + 0.3 * Math.sin(t * 7);
};

/**
 * Docs `reference/visualize/line-plots` (Compare ranges): a plot of three hotfires on
 * three days shows each one at its real time. A click on Relative draws them from one
 * zero in three colors, and a hover over a range name in the legend dims the others.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const time = await client.channels.create({
      name: "hotfire_time",
      isIndex: true,
      dataType: "timestamp",
    });
    const data = await client.channels.create({
      name: CHANNEL,
      dataType: "float32",
      index: time.key,
    });
    const now = TimeStamp.now();
    // One write at a time: concurrent writers to the same channels conflict.
    const created: ranger.New[] = [];
    for (const { name, daysAgo, peak, delay } of TESTS) {
      const start = now.sub(TimeSpan.days(daysAgo));
      const times = new BigInt64Array(SAMPLES);
      const values = new Float32Array(SAMPLES);
      for (let i = 0; i < SAMPLES; i++) {
        times[i] = BigInt(start.add(PERIOD.mult(i)).valueOf());
        values[i] = pressure(i * PERIOD.seconds, peak, delay);
      }
      await client.write(start, { [time.key]: times, [data.key]: values });
      created.push({
        name,
        timeRange: new TimeRange(start, start.add(PERIOD.mult(SAMPLES))),
      });
    }
    const ranges = await client.ranges.create(created);
    const { page } = session;
    await openProject(session, PROJECT);
    const project = (await client.projects.retrieve({ searchTerm: PROJECT })).find(
      (p) => p.name === PROJECT,
    );
    if (project == null) throw new Error(`no project named "${PROJECT}"`);
    await client.lineplots.create({
      name: PLOT,
      project: project.key,
      channels: { y1: [data.key] },
      ranges: ranges.map(({ key }) => ({ variant: "key", key })),
    });
    await capture.clearPanel(session);
    await capture.searchPalette(session, PLOT);
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.showVisualizationToolbar(session);
    await session.settleWall(1500);
    await session.settle(800);
    await session.moveTo(capture.PARK);

    const mode = page.locator(".console-line-plot__toolbar-data-x").first();
    const relative = mode.getByText("Relative", { exact: true }).first();
    const legend = page.locator(".pluto-legend-grouped").first();

    session.startRecording();
    await session.hold(1800);

    await capture.zoomCenter(session, mode, 1.5);
    await session.click(relative, { zoom: false });
    session.endZoom();
    await session.hold(2500);

    await session.moveTo(legend.getByText(TESTS[1].name, { exact: true }).first(), {
      text: true,
    });
    await session.hold(2200);
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await client.close();
  }
};
