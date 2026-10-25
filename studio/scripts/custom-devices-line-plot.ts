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
 * Docs `client/resources/build-device-driver/line-plot`: from an empty panel, create
 * a line plot and add `arduino_value`, which the device script records live.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.arduino();
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    await session.hold(800);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await session.hold(600);
    await capture.showVisualizationToolbar(session);
    await session.zoom(page.locator("label").filter({ hasText: "Y1" }).first(), 1.5);
    await session.hold(500);
    await capture.addChannels(session, "Y1", [fixture.value], {
      search: fixture.value,
    });
    await session.hold(600);
    session.endZoom();
    await capture.hideBottomToolbar(session);
    await session.moveTo({ x: 800, y: 870 });
    // The first seconds of the plot can show a gap in the data. Wait in wall time until
    // the 30 s window scrolls past it.
    await session.settleWall(30000);
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
