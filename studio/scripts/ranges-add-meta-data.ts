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
 * Docs `console/ranges/add-meta-data`: on a range's overview page, click the
 * "+" in the Metadata section, type a key and a value, and commit; the pair
 * appears as a saved row.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedRanges(["Hotfire 09"]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.searchPalette(session, "Hotfire 09");
  const section = page.locator(".pluto-header").filter({ hasText: "Metadata" });
  await session.waitFor(section.first());
  await session.moveTo({ x: 756, y: 400 });

  session.startRecording();
  await session.hold(1000);

  await session.click(section.locator("button").first());
  const key = page.getByPlaceholder("Key").first();
  await session.waitFor(key);
  await session.hold(600);

  await session.click(key);
  await session.type("test_stand");
  await session.click(page.getByPlaceholder("Value").first());
  await session.type("TS-2");
  await session.hold(400);
  await session.press("Enter");

  await session.waitFor(page.getByText("test_stand", { exact: true }).first());
  await session.hold(2200);
};
