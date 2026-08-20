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
 * Docs `console/ranges/palette`: search a saved range by name in the palette
 * and select it; the range loads into the Ranges toolbar and its overview
 * opens.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedRanges(["Hotfire 09", "Coldflow 22", "Burst Test 04"]);
  const { page } = session;
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 400);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1000);

  await capture.searchPalette(session, "Coldflow 22");
  const drawer = page.locator(".console-nav__drawer").first();
  await session.waitFor(drawer.getByText("Coldflow 22", { exact: true }).first());
  await session.waitFor(capture.tab(page, "Coldflow 22"));
  await session.hold(2600);
};
