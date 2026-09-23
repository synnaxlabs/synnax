// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const PARENT = "Dev Qualification 04";
const CHILD = "Hotfire 04-01";
const CHILD_LABELS = ["Hotfire", "Energized", "DQ-4"];

// Today at the given time, matching the modal's default date on camera.
const at = (h: number, m: number, s = 0): TimeStamp => {
  const d = new Date();
  d.setHours(h, m, s, 0);
  return new TimeStamp(d);
};

const LABELS = [
  { name: "Nominal", color: "#3B7DDB" },
  { name: "Anomaly", color: "#DC1360" },
  { name: "DQ-1", color: "#8B5CF6" },
  { name: "DQ-2", color: "#8B5CF6" },
  { name: "DQ-3", color: "#8B5CF6" },
  { name: "DQ-4", color: "#8B5CF6" },
  { name: "Hotfire", color: "#FACC15" },
  { name: "Energized", color: "#FACC15" },
  { name: "Safed", color: "#10B061" },
  { name: "Hold", color: "#F97316" },
];

interface Time {
  h: number;
  m: number;
  s: number;
  ms: number;
}

/**
 * Docs `client/ranges/create-child`: the Dev Qualification tree is staged
 * off-camera, then from the Range explorer the viewer creates Hotfire 04-01 as
 * its child through the command palette, with a time window and labels.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;

  // Off-camera: stage the labels and the full Dev Qualification 04 tree so the
  // shot only shows the final child creation. Existing entities are left alone.
  const keys = await fixtures.ensureLabels(LABELS);
  // The explorer lists each stage in key order, so hand out keys that sort in
  // creation (chronological) order.
  let ordinal = 0;
  const orderedKey = (): string =>
    `${(ordinal++).toString(16).padStart(8, "0")}-0000-4000-8000-000000000000`;
  const rangeKey: Record<string, string> = {
    [PARENT]: await fixtures.ensureRange({
      name: PARENT,
      key: orderedKey(),
      timeRange: new TimeRange(at(0, 0, 0), at(23, 59, 59)),
      labels: [keys["DQ-4"]],
    }),
  };
  // Add a child range; parent defaults to Dev Qualification 04.
  const add = async (
    n: string,
    from: TimeStamp,
    to: TimeStamp,
    labelNames: string[],
    parentName: string = PARENT,
  ): Promise<void> => {
    rangeKey[n] = await fixtures.ensureRange({
      name: n,
      key: orderedKey(),
      timeRange: new TimeRange(from, to),
      parent: rangeKey[parentName],
      labels: labelNames.map((l) => keys[l]),
    });
  };

  const safedNom = ["Safed", "Nominal", "DQ-4"];
  const enerNom = ["Energized", "Nominal", "DQ-4"];

  // The story of the test day; each range starts 1s after the last one ends.
  // Setup runs until the hotfire and nests the verification, ops, and load
  // phases.
  await add("Idle", at(0, 0), at(7, 15), safedNom);
  await add("Setup", at(7, 15, 1), at(13, 0), safedNom);
  await add("Stand Verification", at(7, 15, 2), at(9, 30), safedNom, "Setup");
  await add("Eng Ops", at(9, 30, 1), at(11, 30), enerNom, "Setup");
  await add("Propellant Load", at(11, 30, 1), at(13, 0), enerNom, "Setup");

  await capture.login(session, { username: "synnax", password: "seldon" });

  // Off-camera: show the DQ-4 view in the Range explorer tab.
  await page.getByText("DQ-4", { exact: true }).first().click();
  const explorer = page.locator(".console-range-explorer");
  await session.waitFor(explorer.first());
  await session.moveTo({ x: 640, y: 380 });

  // The overview's own title input also has the "Name" placeholder; scope to
  // the modal or the first match dismisses it.
  const modal = page.locator(".console-modal");
  const name = modal.getByPlaceholder("Name").first();
  const labels = modal.getByText("Select labels", { exact: true }).first();
  const labelItem = (n: string): ReturnType<typeof page.locator> =>
    page
      .locator(".pluto-select__dialog:visible .pluto-list__item")
      .filter({ has: page.getByText(n, { exact: true }) })
      .first();
  const calendars = modal.locator(
    "[class*='time-range'] button:has(.pluto-icon--calendar)",
  );

  // Set a From/To field through its calendar picker: the Hour/Minute/Second
  // lists set the time directly, avoiding the datetime input's text parsing.
  const setTime = async (calendar: typeof calendars, t: Time): Promise<void> => {
    await session.click(calendar);
    const picker = page.locator(".pluto-datetime-modal:visible").first();
    await session.waitFor(picker);
    const lists = picker.locator(".pluto-time-list");
    const item = (which: number, n: number): typeof calendars =>
      lists
        .nth(which)
        .locator(".pluto-list__item")
        .filter({ has: page.getByText(String(n), { exact: true }) })
        .first();
    // Zero the millisecond and second fields instantly as the picker opens, so
    // the viewer only watches the cursor set the values that matter. The second
    // is only set on camera when it is not zero.
    await picker.locator(".pluto-time-selector input").first().fill(String(t.ms));
    await item(2, 0).click();
    const pick = async (which: number, n: number): Promise<void> => {
      await item(which, n)
        .scrollIntoViewIfNeeded()
        .catch(() => {});
      await session.click(item(which, n));
    };
    await pick(0, t.h);
    await pick(1, t.m);
    if (t.s !== 0) await pick(2, t.s);
    await session.hold(80);
    await session.click(picker.getByText("Done", { exact: true }).first());
    await session.hold(80);
  };

  const createRange = async (
    rangeName: string,
    from: Time,
    to: Time,
    parentName: string,
    labelNames: string[],
  ): Promise<void> => {
    // Open the create modal from the command palette. Palette input dispatches
    // natively because rig-stepped typing leaves the list unfiltered.
    const paletteBtn = page.locator(".console-palette button").first();
    await session.moveTo(paletteBtn);
    await session.hold(300);
    await session.click(paletteBtn);
    const input = page.locator(".console-palette__input input[role='textbox']");
    await session.waitFor(input);
    await session.hold(100);
    await session.moveTo(input);
    await session.hold(120);
    for (const ch of ">Create range") {
      await input.press(ch === " " ? "Space" : ch);
      await session.hold(40);
    }
    await session.hold(150);
    const command = page
      .locator(".pluto-list__item")
      .filter({ has: page.getByText("Create range", { exact: true }) })
      .first();
    await session.waitFor(command);
    await session.moveTo(command, { text: true });
    await command.click();
    await session.waitFor(name);
    await session.hold(80);

    await session.click(name);
    await session.type(rangeName, 40);
    await session.hold(80);

    await setTime(calendars.first(), from);
    await setTime(calendars.last(), to);

    // Pick the parent. Linger here; it is the point of the shot.
    const parentTrigger = modal
      .locator(".console-range-create-layout__parent .pluto-dialog__trigger")
      .first();
    await session.moveTo(parentTrigger);
    await session.hold(200);
    await session.click(parentTrigger);
    const parentSearch = page.locator(".pluto-select__dialog:visible input").first();
    await session.waitFor(parentSearch);
    await session.hold(300);
    await session.moveTo(parentSearch);
    await session.hold(200);
    for (const ch of parentName) {
      await parentSearch.press(ch === " " ? "Space" : ch);
      await session.hold(40);
    }
    await session.hold(400);
    const parentItem = page
      .locator(".pluto-select__dialog:visible .pluto-list__item")
      .filter({ has: page.getByText(parentName, { exact: true }) })
      .first();
    await session.waitFor(parentItem);
    await session.hold(300);
    await session.click(parentItem.getByText(parentName, { exact: true }).first(), {
      text: true,
    });
    await session.hold(500);

    // Add each label. Type the name into the dropdown search on camera (native
    // press filters where rig typing does not), then click the single result.
    // The multi-select stays open across picks.
    await session.click(labels, { text: true });
    const search = page.locator(".pluto-select__dialog:visible input").first();
    await session.waitFor(search);
    for (const [i, n] of labelNames.entries()) {
      for (const ch of n) {
        await search.press(ch === " " ? "Space" : ch);
        await session.hold(40);
      }
      await session.hold(200);
      const it = labelItem(n);
      await session.waitFor(it);
      await session.hold(150);
      await session.click(it.getByText(n, { exact: true }).first(), { text: true });
      await session.hold(250);
      // The search keeps its text after a pick, so clear it for the next name.
      if (i < labelNames.length - 1) await search.fill("");
    }
    await session.press("Escape");

    await capture.clickButton(session, "Save to Core");
    await session.waitForHidden(name);
  };

  session.startRecording();
  await session.hold(180);

  // Hotfire 04-01: 1:00:01pm to 1:10:00pm, under Dev Qualification 04.
  await createRange(
    CHILD,
    { h: 13, m: 0, s: 1, ms: 0 },
    { h: 13, m: 10, s: 0, ms: 0 },
    PARENT,
    CHILD_LABELS,
  );
  await session.waitFor(page.getByText(CHILD, { exact: true }).first());
  await session.hold(700);

  // End on the new range's own page. Settle on the row before opening it.
  const row = explorer.getByText(CHILD, { exact: true }).first();
  await session.moveTo(row, { text: true });
  await session.hold(1000);
  await session.click(row, { text: true });
  await session.waitFor(
    page.locator(".pluto-header").filter({ hasText: "Child ranges" }).first(),
  );
  // Rest the cursor on the parent breadcrumb.
  const parentLink = page.locator(".console-range-overview__parent-button").first();
  await session.moveTo(parentLink, { text: true });
  await session.hold(3000);
};
