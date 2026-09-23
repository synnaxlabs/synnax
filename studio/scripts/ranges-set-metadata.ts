// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

// Friday 2025-03-14 10:00 local, ~18 months back, so the overview shows a clean
// dated span instead of a live-clock one.
const START = new TimeStamp(new Date(2025, 2, 14, 10, 0, 0));

/**
 * Docs `client/ranges/set-metadata`: on a range's overview page, click the
 * "+" in the Metadata section, type a key and a value, and commit; the pair
 * appears as a saved row.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createRanges([
    {
      name: "My Range",
      timeRange: new TimeRange(START, START.add(TimeSpan.minutes(15))),
    },
  ]);
  const { page } = session;
  await capture.login(session);

  await capture.searchPalette(session, "My Range");
  const section = page.locator(".pluto-header").filter({ hasText: "Metadata" });
  await session.waitFor(section.first());

  // Script-local override: drop the Snapshots section so the shot stays on the
  // Metadata block.
  await page.evaluate(() => {
    for (const el of document.querySelectorAll(".pluto-header"))
      if (el.textContent?.trim() === "Snapshots")
        el.parentElement?.style.setProperty("display", "none");
  });

  await session.moveTo({ x: 756, y: 400 });

  session.startRecording();
  await session.hold(1000);

  const pairs: [string, string][] = [
    ["test_configuration", "Test 1"],
    ["part_number", "12345"],
  ];

  for (const [k, v] of pairs) {
    await session.click(section.locator("button").first());
    const key = page.getByPlaceholder("Key").first();
    await session.waitFor(key);
    await session.hold(400);

    await session.click(key);
    await session.type(k);
    await session.click(page.getByPlaceholder("Value").first());
    await session.type(v);
    await session.hold(200);
    await session.press("Enter");

    await session.waitFor(page.getByText(k, { exact: true }).first());
    await session.hold(200);
  }
  await session.hold(1000);
};
