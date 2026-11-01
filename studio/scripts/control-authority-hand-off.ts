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

const AUTOMATION = "Valve Guard";

/** Holds the valve closed at authority 100, below the schematic's 200. */
const PROGRAM = `import time

authority 100

func close_valve() {
    valve_cmd = 0
}

time.interval(period=500ms) -> close_valve()
`;

/**
 * Docs `control/authority/hand-off`: an Arc automation holds a valve closed. The
 * operator acquires control from a schematic with higher authority, opens the valve,
 * and releases control, which hands the valve back to the Arc, and it closes again.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const valve = await fixtures.echoValve();
  const automation = await fixtures.runArc({ name: AUTOMATION, text: PROGRAM });
  try {
    const { page } = session;
    const symbol = await createValveSchematic(session, valve, { authority: 200 });
    const entry = (name: string) =>
      page.locator(".pluto-legend-entry").filter({ hasText: name }).first();
    await session.waitFor(entry(AUTOMATION));
    await session.waitFor(symbol.locator(".pluto-control-chip svg.lyra-icon--square"));
    const toggle = symbol.locator("button.pluto-symbol-primitive-toggle").first();
    // The valve shows its echoed state channel, so it reads open only once the echo
    // lands, and closed again once the Arc's next write does.
    const open = symbol.locator("button.pluto-symbol-primitive-toggle.pluto--enabled");
    const control = capture.control(page, "circle");

    session.startRecording();
    await session.hold(1800);

    await session.moveTo(control);
    await session.waitFor(page.getByText("Acquire control", { exact: true }).first());
    await session.hold(900);
    await session.click(control, { zoom: false });
    await session.waitFor(entry("Schematic"));
    await session.hold(1200);

    // The valve's rect alone would zoom past the render's resolution.
    await session.zoom(symbol, 1.6);
    await session.hold(800);
    await session.click(toggle);
    await session.waitFor(open);
    const body = await symbol.boundingBox();
    if (body == null) throw new Error("valve is hidden");
    await session.moveTo({
      x: body.x + body.width + 120,
      y: body.y + body.height + 60,
    });
    await session.hold(1400);
    session.endZoom();

    await session.click(control, { zoom: false });
    await session.waitFor(entry(AUTOMATION));
    await session.waitForHidden(open);
    await session.moveTo({
      x: body.x + body.width + 160,
      y: body.y + body.height + 90,
    });
    await session.hold(3000);
  } finally {
    await automation.stop();
    await valve.stop();
  }
};
