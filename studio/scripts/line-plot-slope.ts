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
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideVisualizationToolbar(session);
    // The measurement pins to data on screen, so stop the rolling window first.
    // A hold snapshots the axis as it turns on, so wait for real samples first;
    // otherwise it pins the empty hour-wide default window.
    await session.settleWall(4000);
    await session.settle(1000);
    await session.click(capture.control(page, "pause"), { zoom: false });
    await session.settle(500);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1600);

    // No zoom on any of these: the controls sit in a corner and the camera
    // must hold the plot.
    await session.click(capture.control(page, "rule"), { zoom: false });
    await session.hold(800);
    await session.click({ x: 560, y: 400 }, { zoom: false });
    await session.hold(1200);

    await session.click(page.locator(".console-control__measure button").last(), {
      zoom: false,
    });
    await session.hold(600);
    await session.click({ x: 1020, y: 620 }, { zoom: false });
    await session.hold(900);
    // The hover tooltip follows the cursor and covers the second readout.
    await session.moveTo({ x: 520, y: 980 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
