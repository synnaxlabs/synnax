// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

/**
 * zoomCenter zooms on the center of a region wider than a zoomed frame. A rect that
 * wide caps the zoom at 1x, so the camera aims at the center and pans with the cursor.
 */
const zoomCenter = async (
  session: capture.CaptureSession,
  region: Locator,
  amount: number,
): Promise<void> => {
  const box = await region.boundingBox();
  if (box == null) throw new Error("zoom region is hidden");
  await session.zoom({ x: box.x + box.width / 2, y: box.y + box.height / 2 }, amount);
};

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
    await session.moveTo({ x: 900, y: 24 });

    session.startRecording();
    await session.hold(1800);

    await capture.showVisualizationToolbar(session);
    await session.hold(400);
    await zoomCenter(
      session,
      page.locator(".console-nav__drawer.pluto--location-bottom").first(),
      1.4,
    );
    await session.moveTo(page.locator("label").filter({ hasText: "Y1" }).first());
    await session.hold(2000);
    session.endZoom();
    await session.moveTo({ x: 900, y: 24 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
