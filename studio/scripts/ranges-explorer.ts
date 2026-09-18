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
 * Docs `console/ranges/explorer`: open the Range Explorer from the Ranges Toolbar,
 * turn on editing, and search it down to the matching ranges.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedRanges([
    "Hotfire 09",
    "Coldflow 22",
    "Hotfire 10",
    "Burst Test 04",
    "Leak Check 03",
  ]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 400);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1000);

  // The link swaps the mosaic to the explorer tab, so the camera stays wide.
  await session.click(page.getByText("Open range explorer", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitFor(page.getByText("Leak Check 03", { exact: true }).first());
  await session.hold(1200);

  // The explorer shows its search field only while editing is on.
  await session.click(page.locator("button:has(.pluto-icon--edit)").first());
  const search = page.getByPlaceholder("Search ranges...").first();
  await session.waitFor(search);
  await session.hold(600);
  await session.click(search);
  await session.type("Hotfire");
  await session.waitForHidden(page.getByText("Coldflow 22", { exact: true }).first());
  await session.hold(2500);
};
