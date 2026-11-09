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
 * Docs `reference/visualize/schematics` (See the channels and last sample of a symbol):
 * with editing off, move the pointer over a valve to show its tooltip, then hide the
 * tooltips from the controls of the schematic and show them again.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const valve = await fixtures.echoValve();
  try {
    const { page } = session;
    const symbol = await createValveSchematic(session, valve);
    await session.click(capture.control(page, "edit-off"), { zoom: false });
    await session.settle(500);
    const box = await symbol.boundingBox();
    if (box == null) throw new Error("valve is hidden");
    const away = { x: box.x + box.width + 300, y: box.y + box.height + 160 };
    await session.moveTo(away);

    session.startRecording();
    await session.hold(1800);

    await session.moveTo(symbol);
    await session.waitFor(page.getByText("Sample age").first());
    await session.zoom(symbol, 1.6);
    await session.hold(3000);
    session.endZoom();

    await session.click(capture.control(page, "tooltip"), { zoom: false });
    await session.hold(600);
    await session.moveTo(symbol);
    await session.hold(1500);
    await session.click(capture.control(page, "tooltip"), { zoom: false });
    await session.hold(600);
    await session.moveTo(symbol);
    await session.waitFor(page.getByText("Sample age").first());
    await session.hold(3000);
  } finally {
    await valve.stop();
  }
};
