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
 * Docs `console/line-plots/axes-tab`: in the Axes tab, pin the Y1 bounds and
 * name the axis; the plot rescales and the label appears.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await capture.login(session, { username: "synnax", password: "seldon" }, project);

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await session.moveTo({ x: 756, y: 400 });

    session.startRecording();
    await session.hold(1200);

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    await session.click(drawer.getByText("Axes", { exact: true }).first(), {
      text: true,
    });
    // The tab opens on X1, the time axis; the shot pins the value axis.
    await session.click(drawer.getByText("Y1", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    const lower = capture.field(page, "Lower bound");
    await session.waitFor(lower);
    await session.hold(600);

    // No zoom on any of these: the plot must stay in frame so the axes
    // rescale on camera.
    await session.click(lower, { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("0");
    await session.press("Enter");
    await session.hold(800);

    await session.click(capture.field(page, "Upper bound"), { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("100");
    await session.press("Enter");
    await session.hold(1200);

    await session.click(capture.field(page, "Label"), { zoom: false });
    await session.type("Pressure (psi)");
    await session.press("Enter");
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};
