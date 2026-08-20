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
 * Docs `console/schematics/align-items`: drag a selection box around three
 * symbols sitting at different heights, then click the align button; they
 * snap onto one line.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  const gate = await capture.place(session, "Gate", { x: 420, y: 200 });
  const ball = await capture.place(session, "Ball", { x: 730, y: 420 });
  const needle = await capture.place(session, "Needle", { x: 1040, y: 260 });
  await capture.deselect(session, { x: 200, y: 600 });
  await session.moveTo({ x: 300, y: 620 });

  session.startRecording();
  await session.hold(1200);

  await capture.selectSymbols(session, [gate, ball, needle]);
  await session.hold(1000);

  // Aligning vertically gives every symbol the same center height, so the
  // three land on one horizontal line.
  const align = capture.toolbarButton(page, "align-y-center");
  await session.waitFor(align);
  await session.click(align, { zoom: false });
  await session.hold(800);

  await session.moveTo({ x: 300, y: 620 });
  await session.hold(2600);
};
