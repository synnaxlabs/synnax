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
 * Docs `console/ui-overview/visualization-toolbar`: with a streaming line
 * plot open, toggle the bottom visualization toolbar, switch to Properties,
 * and rename the plot; the tab title updates live.
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
    await session.moveTo({ x: 756, y: 400 });

    session.startRecording();
    await session.hold(1200);

    await capture.showVisualizationToolbar(session);
    await session.hold(1000);

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    await session.click(drawer.getByText("Properties", { exact: true }).first(), {
      text: true,
    });
    const title = capture.field(page, "Title");
    await session.waitFor(title);
    await session.hold(600);

    // The input selects its text on focus, so typing replaces the old title.
    // No zoom: the plot must stay in frame so the rename shows up live.
    await session.click(title, { zoom: false });
    await session.type("Coolant Loop");
    await session.waitFor(capture.tab(page, "Coolant Loop"));
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};
