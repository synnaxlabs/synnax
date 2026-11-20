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

const PARENT = "Hotfire 05";
const CHILD = "Ignition";
const LABELS = [
  { name: "Hotfire", color: "#FACC15" },
  { name: "Nominal", color: "#3B7DDB" },
];

/**
 * Docs `client/ranges/create-child`: right-click a range in the range explorer, select
 * Start child range, name the child, and create it. The dialog fills in the parent, and
 * the child appears under it in the range explorer.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const keys = await fixtures.ensureLabels(LABELS);
  const now = TimeStamp.now();
  await fixtures.ensureRange({
    name: PARENT,
    timeRange: new TimeRange(now.sub(TimeSpan.hours(1)), now.add(TimeSpan.hours(1))),
    labels: [keys.Hotfire],
  });
  await capture.login(session);
  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 400);
  await page.getByRole("button", { name: "Open range explorer", exact: true }).click();
  // The parent runs, so the toolbar lists it too.
  const explorer = page.locator(".console-range-tree__list").first();
  const parent = explorer.getByRole("listitem", { name: PARENT, exact: true }).first();
  await session.waitFor(parent);
  await session.settle(500);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, parent, "Start child range");
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Range name").first();
  await session.waitFor(name);
  await session.hold(400);
  await session.zoom(modal);
  // The dialog fills in the parent, so rest on it before naming the child.
  await session.moveTo(modal.getByText(PARENT, { exact: true }).first(), {
    text: true,
  });
  await session.hold(1000);

  await session.click(name);
  await session.type(CHILD);
  await session.hold(300);

  const labels = modal.getByRole("button", { name: "Labels", exact: true }).first();
  await session.click(labels, { text: true });
  const item = page
    .locator(".lyra-dialog__dialog")
    .filter({ has: page.getByPlaceholder("Search labels...") })
    .locator(".lyra-list__item")
    .filter({ has: page.getByText("Nominal", { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(400);
  await session.click(item, { text: true });
  await session.hold(1000);

  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  // The child lands under its parent, which stays closed.
  await session.click(parent.getByRole("button", { name: "Expand" }));
  await session.waitFor(
    explorer.getByRole("listitem", { name: CHILD, exact: true }).first(),
  );
  await session.moveTo({ x: 1000, y: 600 });
  await session.hold(3000);
};
