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

const ABSOLUTE = "You have absolute control. Click to release.";

/**
 * Docs `control/manual/absolute-control`: acquire control of a schematic, then click
 * the control chip beside a valve to take absolute control of its channel. The chip
 * turns green, and its tooltip says what that means.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const valve = await fixtures.echoValve();
  try {
    const { page } = session;
    const symbol = await createValveSchematic(session, valve);
    const chip = symbol.locator(".pluto-control-chip").first();

    session.startRecording();
    await session.hold(1800);

    await session.click(capture.control(page, "circle"), { zoom: false });
    await session.waitFor(page.locator(".pluto-legend-entry").first());
    await session.hold(1000);

    // The valve's rect alone would zoom past the render's resolution.
    await session.zoom(symbol, 1.6);
    await session.hold(800);
    await session.click(chip);
    session.endZoom();
    // Only absolute control gives the chip a background.
    await session.waitFor(symbol.locator(".pluto-control-chip[style*='secondary']"));
    const body = await symbol.boundingBox();
    if (body == null) throw new Error("valve is hidden");
    // Off the chip so the green reads, then back on it for a fresh tooltip.
    await session.moveTo({ x: body.x + body.width + 90, y: body.y + body.height / 2 });
    await session.hold(1000);
    await session.moveTo(chip);
    await session.waitFor(page.getByText(ABSOLUTE, { exact: true }).first());
    await session.hold(3000);
  } finally {
    await valve.stop();
  }
};
