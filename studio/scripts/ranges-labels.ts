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

const NEW_LABEL = "Hotfire 04-09";
/** The green Nominal used to carry, reused for the new label. */
const GREEN = "10B061";
// Friday 2025-03-14 10:00 local, matching the set-metadata shot's My Range.
const START = new TimeStamp(new Date(2025, 2, 14, 10, 0, 0));

/**
 * Docs `client/ranges/labels`: on a range's overview page, create a new green
 * label through "New label", select it, then select a pre-made label.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // Start from a bare workspace: only the Nominal and Anomaly labels survive.
  await fixtures.clearWorkspace();
  await fixtures.resetLabels([
    { name: "Nominal", color: "#3B7DDB" },
    { name: "Anomaly", color: "#DC1360" },
  ]);
  await fixtures.seedRanges([
    {
      name: "My Range",
      timeRange: new TimeRange(START, START.add(TimeSpan.minutes(15))),
    },
  ]);
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.searchPalette(session, "My Range");
  const trigger = page.getByText("Select labels", { exact: true }).first();
  const listItem = (name: string) =>
    page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: name })
      .first();
  // No labels are selected until the end, so the placeholder trigger opens the
  // selector every time.
  const openSelector = (): Promise<void> => session.click(trigger, { text: true });
  const pick = async (name: string): Promise<void> => {
    const item = listItem(name);
    await session.waitFor(item);
    await session.hold(250);
    await session.click(item.getByText(name, { exact: true }).first(), { text: true });
    await session.hold(250);
  };

  await session.waitFor(trigger);

  // Script-local override: drop the sections unrelated to labels so the shot
  // stays on the Details block.
  await page.evaluate(() => {
    const hide = ["Child ranges", "Metadata", "Snapshots"];
    for (const el of document.querySelectorAll(".pluto-header"))
      if (hide.includes(el.textContent?.trim() ?? ""))
        (el.parentElement as HTMLElement | null)?.style.setProperty("display", "none");
  });

  await session.moveTo({ x: 756, y: 400 });

  session.startRecording();
  await session.hold(1000);

  // Create the new label through the selector's "New label" button.
  await openSelector();
  const newLabel = page
    .locator(".pluto-dialog__dialog")
    .getByText("New label", { exact: true })
    .first();
  await session.waitFor(newLabel);
  await session.hold(400);
  await session.click(newLabel, { text: true });

  // The label editor opens; start a fresh label from its own "New label" row.
  const modal = page.locator(".console-label__edit");
  await session.waitFor(modal);
  await session.hold(400);
  await session.click(modal.getByText("New label", { exact: true }).first(), {
    text: true,
  });

  const nameInput = modal.getByPlaceholder("Name").first();
  await session.waitFor(nameInput);
  await session.click(nameInput);
  await session.type(NEW_LABEL, 40);
  await session.hold(300);

  // Set the color by hex, then close the picker so the swatch commits on blur.
  const swatch = modal.locator(".pluto-color-swatch").first();
  await session.click(swatch);
  const hex = page.locator(".pluto-color-picker input").first();
  await session.waitFor(hex);
  await session.hold(200);
  await session.click(hex);
  await session.hold(200);
  await session.press("ControlOrMeta+a");
  await session.hold(200);
  await session.press("Delete");
  await session.hold(200);
  await hex.fill(GREEN);
  await session.hold(300);
  await session.click(swatch);
  await session.hold(300);

  await session.click(modal.locator("button:has(.pluto-icon--check)").first());
  await session.hold(400);
  await session.click(modal.locator("button:has(.pluto-icon--close)").first());
  await session.hold(300);

  // Open the selector once and add both labels; multi-select stays open.
  await openSelector();
  await pick(NEW_LABEL);
  await pick("Nominal");
  await session.press("Escape");
  await session.waitFor(
    page.locator(".pluto-tag").filter({ hasText: NEW_LABEL }).first(),
  );
  await session.waitFor(
    page.locator(".pluto-tag").filter({ hasText: "Nominal" }).first(),
  );
  await session.hold(1500);
};
