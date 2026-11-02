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
    // Off the plot, so no hover tooltip covers it.
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await capture.showVisualizationToolbar(session);
    await session.hold(400);
    await capture.zoomCenter(
      session,
      page.locator(".console-nav__drawer.lyra--location-bottom").first(),
      1.4,
    );
    await session.moveTo(page.locator("label").filter({ hasText: "Y1" }).first());
    await session.hold(2000);
    session.endZoom();
    await session.moveTo(capture.PARK);
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
