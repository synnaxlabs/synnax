// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { capture, fixtures } from "@/index";

const WINDOW = "Test window";
const PARENT = "Hotfire 12";
const CHILD = "Ignition";
const SAMPLES = 1200;
const PERIOD = TimeSpan.seconds(1);

// A smooth step from 0 to 1 between a and b.
const ramp = (t: number, a: number, b: number): number => {
  const x = Math.min(Math.max((t - a) / (b - a), 0), 1);
  return x * x * (3 - 2 * x);
};

/**
 * Still for docs image `console/ranges/annotations`: a line plot of one hotfire, with
 * the Hotfire 12 range and its Ignition child drawn as range annotations. The last
 * captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  const start = TimeStamp.now().sub(TimeSpan.hours(2));
  const at = (i: number): TimeStamp => start.add(PERIOD.mult(i));
  try {
    const time = await client.channels.create(
      { name: "hotfire_time", isIndex: true, dataType: "timestamp" },
      { retrieveIfNameExists: true },
    );
    const [chamber, tank] = await client.channels.create(
      ["chamber_pressure", "tank_pressure"].map((name) => ({
        name,
        dataType: "float32",
        index: time.key,
      })),
      { retrieveIfNameExists: true },
    );
    const times = new BigInt64Array(SAMPLES);
    const chamberData = new Float32Array(SAMPLES);
    const tankData = new Float32Array(SAMPLES);
    for (let i = 0; i < SAMPLES; i++) {
      times[i] = BigInt(at(i).valueOf());
      const noise = Math.sin(i * 1.7) + 0.6 * Math.sin(i * 4.3);
      const pressurized = ramp(i, 380, 440) - ramp(i, 860, 940);
      const burn = ramp(i, 540, 560) - ramp(i, 720, 745);
      tankData[i] = 20 + 430 * pressurized - 70 * burn + 2 * noise;
      chamberData[i] = 15 + 520 * burn + (burn > 0.5 ? 6 * noise : noise);
    }
    await client.write(start, {
      [time.key]: times,
      [chamber.key]: chamberData,
      [tank.key]: tankData,
    });

    await client.ranges.create({
      name: WINDOW,
      timeRange: new TimeRange(at(0), at(SAMPLES - 1)),
    });
    // Annotation titles share one strip and draw in key order, so the child's title
    // must draw after its parent's.
    const parent = await client.ranges.create({
      key: "00000001-0000-4000-8000-000000000000",
      name: PARENT,
      color: "#3B7DDB",
      timeRange: new TimeRange(at(360), at(900)),
    });
    await client.ranges.create({
      key: "00000002-0000-4000-8000-000000000000",
      name: CHILD,
      color: "#F97316",
      timeRange: new TimeRange(at(530), at(750)),
      parent: { key: parent.key },
    });
  } finally {
    await client.close();
  }

  const { page } = session;
  await capture.login(session);
  // Opening a range from the palette makes it active, and a new plot opens on it.
  await capture.searchPalette(session, WINDOW);
  await capture.clearPanel(session);
  await capture.clickPanelCreate(session);
  await capture.createComponent(session, "Line plot");
  await session.waitFor(page.locator(".pluto-line-plot").first());

  await capture.addChannels(session, "Y1", ["chamber_pressure", "tank_pressure"], {
    search: "pressure",
  });
  await capture.hideBottomToolbar(session);
  await session.settleWall(3000);
  await session.settle(500);
  // Off the plot, so no crosshair or readout shows.
  await session.moveTo(capture.PARK);

  session.startRecording();
  await session.hold(2500);
};
