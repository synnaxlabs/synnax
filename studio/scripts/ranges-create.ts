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

const LABELS = [
  { name: "Hotfire", color: "#FACC15" },
  { name: "Nominal", color: "#3B7DDB" },
  { name: "Anomaly", color: "#DC1360" },
];

const hoursAgo = (h: number): TimeStamp => TimeStamp.now().sub(TimeSpan.hours(h));

/**
 * Docs `client/ranges/create`: click the "+" in the Ranges toolbar, name the range,
 * add a label, and save it to the Core. The new range appears in the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const keys = await fixtures.ensureLabels(LABELS);
  await fixtures.ensureRange({
    name: "Hotfire 03",
    timeRange: new TimeRange(hoursAgo(30), hoursAgo(29)),
    labels: [keys.Hotfire, keys.Nominal],
  });
  await fixtures.ensureRange({
    name: "Hotfire 04",
    timeRange: new TimeRange(hoursAgo(6), hoursAgo(5)),
    labels: [keys.Hotfire, keys.Anomaly],
  });
  await capture.login(session);
  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 400);
  await page.getByText("Open range explorer", { exact: true }).first().click();
  await session.waitFor(page.locator(".console-range-explorer").first());
  await session.settle(500);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.clickToolbarCreate(session);
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(400);
  await session.zoom(modal);

  await session.click(name);
  await session.type("Hotfire 05");
  await session.hold(300);

  await session.click(modal.getByText("Labels", { exact: true }).first(), {
    text: true,
  });
  const item = page
    .locator(".lyra-dialog__dialog")
    .filter({ has: page.getByPlaceholder("Search labels...") })
    .locator(".lyra-list__item")
    .filter({ has: page.getByText("Hotfire", { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(400);
  await session.click(item, { text: true });
  await session.hold(1000);

  await capture.clickButton(session, "Save to Core");
  session.endZoom();
  await session.waitForHidden(name);
  await session.waitFor(
    page
      .locator(".console-range-explorer")
      .getByText("Hotfire 05", { exact: true })
      .first(),
  );
  await session.hold(3000);
};
