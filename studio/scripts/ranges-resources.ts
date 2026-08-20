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
 * Docs `console/ranges/resources`: browse the Core's ranges in the range
 * explorer, favorite one into the Ranges toolbar, and click it there to make
 * it the active range.
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

  // The link swaps the mosaic to the explorer tab, so the camera stays wide.
  await session.click(
    page.getByText("Open range explorer", { exact: true }).first(),
    { text: true, zoom: false },
  );
  const row = page.getByText("Hotfire 09", { exact: true }).first();
  await session.waitFor(row);
  await session.hold(800);

  await capture.contextMenu(session, row, "Favorite");
  const drawer = page.locator(".console-nav__drawer").first();
  const favorited = drawer.getByText("Hotfire 09", { exact: true }).first();
  await session.waitFor(favorited);
  await session.hold(600);

  await session.click(favorited, { text: true });
  await session.waitFor(
    drawer.locator(".console-range-list-item.pluto--selected").first(),
  );
  await session.hold(2200);
};
