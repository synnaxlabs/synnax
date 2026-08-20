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
 * Docs `console/line-plots/pan`: pick the pan tool and drag across the plot;
 * the view translates along the data.
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
    await capture.hideVisualizationToolbar(session);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1600);

    // No zoom: the controls sit in a corner and the camera must hold the plot.
    await session.click(capture.control(page, "pan"), { zoom: false });
    await session.hold(800);
    await session.drag({ x: 1050, y: 480 }, { x: 560, y: 480 }, { zoom: false });
    await session.hold(2400);
    await session.drag({ x: 560, y: 480 }, { x: 1000, y: 560 }, { zoom: false });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
