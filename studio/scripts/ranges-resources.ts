// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const RANGES = ["Hotfire 09", "Coldflow 22", "Burst Test 04"];

/**
 * Docs `console/ranges/resources`: open the Ranges Toolbar, which lists the
 * favorited ranges, and click one to load it as the active range.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    await fixtures.seedRanges(RANGES);
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    // The plot must exist before any range is favorited: favoriting also makes the
    // range active, and a plot created under an active range inherits its historic
    // window instead of a live rolling one.
    await capture.commandPalette(session, "Create line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);

    // Selecting a range from the palette favorites it into the toolbar and opens an
    // overview tab the shot doesn't want.
    for (const name of RANGES) {
      await capture.searchPalette(session, name);
      await session.waitFor(capture.tab(page, name));
      await capture.closeTab(session, name);
    }
    await capture.openToolbar(session, "range");
    await capture.resizeToolbar(session, 400);
    await capture.closeToolbar(session);
    // The plot's stream buffer restarts each time an overview tab covered it, and it
    // fills on wall time: give it a full rolling window of data.
    await session.settleWall(32000);
    await session.settle(1000);
    await session.moveTo({ x: 900, y: 480 });

    session.startRecording();
    await session.hold(1000);

    await capture.openToolbar(session, "range");
    const drawer = page.locator(".console-nav__drawer").first();
    const favorited = drawer.getByText("Hotfire 09", { exact: true }).first();
    await session.waitFor(favorited);
    await session.hold(1200);

    await session.click(favorited, { text: true });
    await session.waitFor(
      drawer.locator(".console-range-list-item.pluto--selected").first(),
    );
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};
