// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const RANGE = "Hotfire 09";
const CHANNEL = "tank_pressure";

/**
 * Docs `reference/visualize/line-plots` (Ranges on the plot): the plot shows the ranges
 * of a test as bars above it. Hover over a bar to read the range, click it to zoom to
 * the range and pause the plot, and click it again to go back.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createStaticTelemetry({ channels: [CHANNEL] });
  const client = fixtures.connect();
  try {
    // The static block ends 5 seconds before now and spans 120 seconds.
    const now = TimeStamp.now();
    const ago = (seconds: number): TimeStamp => now.sub(TimeSpan.seconds(seconds));
    await client.ranges.create([
      { name: RANGE, timeRange: new TimeRange(ago(125), ago(5)) },
      {
        name: "Pressurization",
        timeRange: new TimeRange(ago(110), ago(80)),
        color: "#3B7DDB",
      },
      {
        name: "Hold",
        timeRange: new TimeRange(ago(65), ago(20)),
        color: "#F18F01",
      },
    ]);
    const { page } = session;
    await capture.login(session);
    // Opening a range from the palette makes it active, and a new plot opens on it.
    await capture.searchPalette(session, RANGE);
    await capture.clearPanel(session);
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", [CHANNEL], { search: CHANNEL });
    await capture.hideBottomToolbar(session);
    await session.settleWall(1500);
    await session.settle(800);
    await session.moveTo(capture.PARK);

    const strip = page.locator(".pluto-lineplot-range-provider").first();
    await session.waitFor(strip);
    const box = await strip.boundingBox();
    if (box == null) throw new Error("the range strip is not on screen");
    // The plot spans the 120 seconds of the range, and Hold runs from 60 to 105 of them.
    const bar = { x: box.x + box.width * 0.7, y: box.y + 14 };

    session.startRecording();
    await session.hold(1800);

    await session.moveTo(bar);
    await session.hold(2200);
    await session.click(bar, { zoom: false });
    await session.hold(2800);
    await session.click(bar, { zoom: false });
    await session.hold(1200);
    await session.moveTo(capture.PARK);
    await session.hold(2500);
  } finally {
    await client.close();
  }
};
