// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { pickComponent } from "./common/create-component";

/**
 * Docs `console/line-plots/create`: open the component selector on an empty panel,
 * create a line plot, press V, and select a channel in Y1 on the Data tab; the plot
 * draws the channel's data.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    // V toggles the toolbar, so it starts closed.
    await capture.hideBottomToolbar(session);
    // The plot reads stored samples when the channel lands, so let some accrue.
    await session.settleWall(10000);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    await pickComponent(session, "Line plot", ".pluto-line-plot");
    await session.hold(1200);

    await session.press("v");
    await session.waitFor(page.locator("label").filter({ hasText: "Y1" }).first());
    await session.hold(800);
    await capture.zoomCenter(
      session,
      page.locator(".console-nav__drawer.lyra--location-bottom").first(),
      1.4,
    );
    const channel = fixture.channels[0];
    await capture.addChannels(session, "Y1", [channel], { search: channel });
    session.endZoom();
    // The new line has a gap where its stream started. It fills on wall time, so wait
    // out a full rolling window.
    await session.settleWall(32000);
    await session.hold(600);

    // Off the plot, so no crosshair covers the new line.
    await session.moveTo(capture.PARK);
    await session.settleWall(1500);
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
