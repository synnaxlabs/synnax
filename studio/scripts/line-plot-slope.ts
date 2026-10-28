// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `console/line-plots/slope`: open the measure tool, click a first point,
 * switch to the second, and click again; the overlay reads out the slope and
 * the deltas.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    // The measurement pins to data on screen, so stop the rolling window first.
    // A hold snapshots the axis as it turns on, so wait for real samples first;
    // otherwise it pins the empty hour-wide default window.
    await session.settleWall(9000);
    await session.settle(1000);
    await session.click(capture.control(page, "pause"), { zoom: false });
    await session.settle(500);
    // The first seconds of data vanish once a measurement lands, so a flat drag
    // zooms the time axis past them.
    const box = await page.locator(".pluto-line-plot").first().boundingBox();
    if (box == null) throw new Error("line plot is hidden");
    const at = (x: number, y: number) => ({
      x: box.x + x * box.width,
      y: box.y + y * box.height,
    });
    await session.drag(at(0.3, 0.7), at(0.97, 0.71), { zoom: false });
    await session.settle(800);
    // Off the plot, so no hover tooltip shows before the first click.
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    // The plot fills the Console, so the camera stays wide.
    await session.click(capture.control(page, "rule"), { zoom: false });
    await session.hold(800);
    await session.click(at(0.3, 0.4), { zoom: false });
    await session.hold(1200);

    await session.click(page.locator(".console-control__measure button").last(), {
      zoom: false,
    });
    await session.hold(600);
    await session.click(at(0.6, 0.35), { zoom: false });
    await session.hold(900);
    // The hover tooltip follows the cursor and covers the second readout.
    await session.moveTo(capture.PARK);
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
