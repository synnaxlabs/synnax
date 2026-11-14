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
 * label through "Create label", which adds it to the range, then select a pre-made
 * label.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // Start from a bare workspace: only the Nominal and Anomaly labels survive.
  await fixtures.clearWorkspace();
  await fixtures.resetLabels([
    { name: "Nominal", color: "#3B7DDB" },
    { name: "Anomaly", color: "#DC1360" },
  ]);
  await fixtures.createRanges([
    {
      name: "My Range",
      timeRange: new TimeRange(START, START.add(TimeSpan.minutes(15))),
    },
  ]);
  const { page } = session;
  await capture.login(session);

  await capture.searchPalette(session, "My Range");
  const trigger = page.getByText(/^(Select labels|Labels)$/).first();
  const listItem = (name: string) =>
    page
      .locator(".lyra-list__item:not(.lyra-tree__item)")
      .filter({ hasText: name })
      .first();
  const pick = async (name: string): Promise<void> => {
    const item = listItem(name);
    await session.waitFor(item);
    await session.hold(250);
    await session.click(item.getByText(name, { exact: true }).first(), { text: true });
    await session.hold(250);
  };

  await session.waitFor(trigger);

  // Script-local override: drop the lines that add other parts, so the shot stays on
  // the header.
  await page.evaluate(() => {
    for (const el of document.querySelectorAll<HTMLElement>(
      ".console-range-overview__add",
    ))
      el.style.setProperty("display", "none");
  });

  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  // Create the new label through the selector's "Create label" button.
  await session.click(trigger, { text: true });
  const create = page
    .locator(".lyra-dialog__dialog")
    .getByText("Create label", { exact: true })
    .first();
  await session.waitFor(create);
  await session.hold(400);
  await session.click(create, { text: true });

  const modal = page.locator(capture.MODAL).first();
  const nameInput = modal.getByPlaceholder("Label name", { exact: true });
  await session.waitFor(nameInput);
  await session.hold(400);
  await session.zoom(modal);

  // Set the color by hex, then close the picker so the swatch commits.
  const swatch = modal.getByLabel("Label color", { exact: true });
  await session.click(swatch);
  const hex = page.locator(".lyra-color-picker").getByLabel("Hex", { exact: true });
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

  await session.click(nameInput);
  await session.type(NEW_LABEL, 40);
  await session.hold(300);
  await session.click(modal.getByRole("button", { name: "Create" }));
  await session.waitForHidden(modal);
  session.endZoom();
  await session.waitFor(
    page.locator(".lyra-tag").filter({ hasText: NEW_LABEL }).first(),
  );
  await session.hold(600);

  // The range holds a label now, so the selector opens from its add button. The
  // header of the range overview holds the label selector.
  await session.zoom(page.locator(".console-range-overview__header").first());
  const labels = page.locator(".lyra-dialog__trigger").filter({ hasText: NEW_LABEL });
  await session.click(labels.locator("button:has(.lyra-icon--add)").first());
  await pick("Nominal");
  session.endZoom();
  await session.press("Escape");
  await session.waitFor(
    page.locator(".lyra-tag").filter({ hasText: NEW_LABEL }).first(),
  );
  await session.waitFor(
    page.locator(".lyra-tag").filter({ hasText: "Nominal" }).first(),
  );
  await session.hold(3000);
};
