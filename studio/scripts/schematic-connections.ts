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
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  const tank = await capture.place(session, "Tank", { x: 520, y: 300 });
  const valve = await capture.place(session, "Gate", { x: 950, y: 300 });
  await capture.deselect(session, { x: 1300, y: 600 });
  await session.moveTo({ x: 756, y: 640 });

  session.startRecording();
  await session.hold(1200);

  // Hovering paints the attachment points; hold on them before the drag.
  await session.moveTo(tank);
  await session.hold(1000);

  await session.drag(capture.handle(tank, "right"), capture.handle(valve, "left"), {
    zoom: false,
  });
  await session.hold(600);

  await session.moveTo({ x: 756, y: 660 });
  await session.hold(2600);
};
