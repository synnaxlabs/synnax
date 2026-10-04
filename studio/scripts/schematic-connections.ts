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
  await capture.deselect(session, { x: 100, y: 290 });
  await capture.fitView(session);
  await session.moveTo({ x: 800, y: 620 });

  session.startRecording();
  await session.hold(1800);

  const [tankBox, valveBox] = await Promise.all([
    tank.boundingBox(),
    valve.boundingBox(),
  ]);
  const view = page.viewportSize();
  if (tankBox == null || valveBox == null || view == null)
    throw new Error("tank or valve is hidden");
  const left = Math.min(tankBox.x, valveBox.x);
  const top = Math.min(tankBox.y, valveBox.y);
  const right = Math.max(tankBox.x + tankBox.width, valveBox.x + valveBox.width);
  const bottom = Math.max(tankBox.y + tankBox.height, valveBox.y + valveBox.height);
  await session.zoom(
    { x: (left + right) / 2, y: (top + bottom) / 2 },
    Math.min(
      1.6,
      view.width / (right - left + 192),
      view.height / (bottom - top + 192),
    ),
  );

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
