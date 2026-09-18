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
 * Docs `console/ranges/add-label`: on a range's overview page, open the label
 * selector and pick a label; the chip appears on the range.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedLabels([
    { name: "Nominal", color: "#10B061" },
    { name: "Anomaly", color: "#DC1360" },
    { name: "Cryo", color: "#3B7DDB" },
  ]);
  await fixtures.seedRanges(["Hotfire 09"]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.searchPalette(session, "Hotfire 09");
  const trigger = page.getByText("Select labels", { exact: true }).first();
  await session.waitFor(trigger);
  await session.moveTo({ x: 756, y: 400 });

  session.startRecording();
  await session.hold(1000);

  await session.click(trigger, { text: true });
  const item = page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
    .filter({ hasText: "Nominal" })
    .first();
  await session.waitFor(item);
  await session.hold(400);
  await session.click(item.getByText("Nominal", { exact: true }).first(), {
    text: true,
  });
  await session.hold(600);
  await session.press("Escape");

  await session.waitFor(
    page.locator(".pluto-tag").filter({ hasText: "Nominal" }).first(),
  );
  await session.hold(2200);
};
