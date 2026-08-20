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
 * Docs `console/ui-overview/range-toolbar`: open the Ranges toolbar from the
 * left rail, show the favorited ranges, and click through one.
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

    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    await capture.closeToolbar(session);

    for (const name of ["Tank Fill", "Hotfire"]) {
      await capture.searchPalette(session, name);
      await session.waitFor(capture.tab(session.page, name));
      await capture.closeTab(session, name);
    }
    await session.settleWall(32000);
    await session.settle(1000);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1000);

    await capture.openToolbar(session, "range");
    await session.hold(1800);

    const drawer = session.page
      .locator(".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)")
      .first();
    await session.click(drawer.getByText("Tank Fill", { exact: true }).first(), {
      text: true,
    });
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};
