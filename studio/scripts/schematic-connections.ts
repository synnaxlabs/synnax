// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

/**
 * Docs `console/schematics/connections`: hover a symbol so its attachment
 * points appear, then drag from one to another symbol's attachment point; the
 * two symbols end up joined by a connection.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  // New symbols drop at the canvas center, so only the last one may sit there.
  const tank = await capture.place(session, "Tank", { x: 300, y: 160 });
  const valve = await capture.place(session, "Gate", { x: 700, y: 160 });
  // The tank's side handle sits off its body's center. Level the valve's handle with
  // it, or the connection draws a step. A node drag drops the pointer travel before its
  // start threshold, so the drag starts sideways and only x loses that travel.
  const [from, to, body] = await Promise.all([
    capture.handle(tank, "right").boundingBox(),
    capture.handle(valve, "left").boundingBox(),
    valve.locator(".pluto-drag-handle").first().boundingBox(),
  ]);
  if (from == null || to == null || body == null)
    throw new Error("the symbols are not on screen");
  const center = { x: body.x + body.width / 2, y: body.y + body.height / 2 };
  const dy = from.y + from.height / 2 - to.y - to.height / 2;
  await session.drag(
    center,
    [
      { x: center.x + 20, y: center.y },
      { x: center.x, y: center.y + dy },
    ],
    { zoom: false },
  );
  await capture.deselect(session, { x: 100, y: 290 });
  await capture.fitView(session);
  await session.moveTo({ x: 800, y: 620 });

  session.startRecording();
  await session.hold(1800);

  await session.zoom([tank, valve], 1.6);

  // Hovering paints the attachment points; hold on them before the drag.
  await session.moveTo(tank);
  await session.hold(1000);

  await session.drag(capture.handle(tank, "right"), capture.handle(valve, "left"), {
    zoom: false,
  });
  session.endZoom();
  await session.hold(600);

  await session.moveTo({ x: 800, y: 620 });
  await session.hold(2800);
};
