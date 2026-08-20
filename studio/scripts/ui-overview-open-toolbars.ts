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
 * Docs `console/ui-overview/open-toolbars`: with a live line plot streaming in
 * the mosaic, open toolbars from the left rail one after another (channels,
 * then ranges), then close the drawer to return to the full plot.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    await fixtures.seedRanges(["Tank Fill", "Hotfire"]);
    await capture.login(session, { username: "synnax", password: "seldon" });

    // The plot must exist before any range is favorited: favoriting also makes
    // the range active, and a plot created under an active range inherits its
    // historic window instead of a live rolling one.
    await capture.commandPalette(session, "Create line plot");
    await session.waitFor(session.page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideVisualizationToolbar(session);

    // Left drawers open at their minimum width, which reads cramped on frame;
    // pre-size the rail to the maximum so recorded opens land wide.
    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    await capture.closeToolbar(session);

    // Favorite the seeded ranges so the Ranges toolbar has entries: selecting a
    // range from the palette adds it to the toolbar and opens its overview tab,
    // which the shot doesn't want.
    for (const name of ["Tank Fill", "Hotfire"]) {
      await capture.searchPalette(session, name);
      await session.waitFor(capture.tab(session.page, name));
      await capture.closeTab(session, name);
    }
    // The plot's stream buffer restarts each time an overview tab covered it,
    // and it fills on wall time: give it a full rolling window of data.
    await session.settleWall(32000);
    await session.settle(1000);
    await session.moveTo({ x: 900, y: 480 });

    session.startRecording();
    await session.hold(1000);

    await capture.openToolbar(session, "channel");
    await session.hold(2200);

    await capture.openToolbar(session, "range");
    await session.hold(2200);

    await capture.closeToolbar(session);
    await session.hold(1500);
  } finally {
    await fixture.stop();
  }
};
