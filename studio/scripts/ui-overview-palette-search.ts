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
 * Docs `console/ui-overview/palette-search`: with a live line plot streaming,
 * open the palette, search for a seeded range by name, and select it; its
 * overview page opens and the range becomes active.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    await fixtures.seedRanges(["Tank Fill", "Hotfire"]);
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await capture.login(session, { username: "synnax", password: "seldon" }, project);

    await capture.commandPalette(session, "Create line plot");
    await session.waitFor(session.page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideVisualizationToolbar(session);
    await session.settleWall(32000);
    await session.settle(1000);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1000);

    await capture.searchPalette(session, "Hotfire");
    await session.waitFor(capture.tab(session.page, "Hotfire"));
    await session.hold(2500);
  } finally {
    await fixture.stop();
  }
};
