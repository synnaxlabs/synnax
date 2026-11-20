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
 * Docs `console/line-plots/toolbar`: with a streaming plot open, click the
 * visualize button in the bottom-left corner; the visualization toolbar opens
 * on the plot's tabs.
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
    // Samples start with the fixture, so the 30-second window fills on wall time.
    await session.settleWall(20000);
    // Off the plot, so no hover tooltip covers it.
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await capture.showVisualizationToolbar(session);
    await session.hold(400);
    // The header's actions and tabs sit at its right end.
    const tabs = page
      .locator(".console-nav__drawer.lyra--location-bottom .lyra-tabs__selector")
      .first();
    await capture.zoomCenter(session, tabs, 1.5);
    await session.moveTo(tabs.getByText("Axes", { exact: true }).first(), {
      text: true,
    });
    await session.hold(2000);
    session.endZoom();
    await session.moveTo(capture.PARK);
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
