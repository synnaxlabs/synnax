// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { createValveSchematic } from "./common/valve-schematic";

/**
 * Docs `console/schematics/valve`: acquire control on a schematic, click the
 * valve to command it open, and watch the control indicator, the legend, and
 * the valve's own state follow the echoed state channel.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.echoValve();
  try {
    const { page } = session;
    const valve = await createValveSchematic(session, fixture);

    session.startRecording();
    await session.hold(1800);

    await session.click(capture.control(page, "circle"), { zoom: false });
    await session.waitFor(page.locator(".pluto-legend-entry").first());
    await session.hold(1400);

    // The valve's rect alone would zoom past the render's resolution.
    await session.zoom(valve, 1.6);
    await session.hold(900);
    await session.click(valve.locator("button.pluto-symbol-primitive-toggle").first());
    session.endZoom();
    await session.hold(400);
    // The echo runs on wall time: command out, state back through the streamer.
    await session.settleWall(600);
    const body = await valve.boundingBox();
    if (body == null) throw new Error("valve is hidden");
    // Clear the cursor off the valve so the indicator stays readable.
    await session.moveTo({
      x: body.x + body.width + 160,
      y: body.y + body.height + 90,
    });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
