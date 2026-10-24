// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const RANGE = "Hotfire 09";
const TASK = "Engine alerts";

/**
 * Docs `console/ranges/snapshot-task`: set a range as the active range, right-click a
 * task in the tasks toolbar and select "Snapshot to" the range, then open the range
 * overview, which lists the task's snapshot.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // A PagerDuty alert task runs on the Core itself, so the shot needs no hardware.
  await fixtures.createStatuses([{ key: "engine-1", name: "Engine 1" }]);
  await fixtures.createAlertTask({ name: TASK, statuses: ["engine-1"] });
  await fixtures.createRanges([RANGE]);
  const { page } = session;
  await capture.login(session);

  // Opening the range from the palette adds it to the ranges toolbar but also makes
  // it active, which the shot does on camera.
  await capture.searchPalette(session, RANGE);
  await capture.clearPanel(session);
  await capture.openToolbar(session, "range");
  const range = page
    .locator(".console-range-list-item")
    .filter({ hasText: RANGE })
    .first();
  await session.waitFor(range);
  await capture.contextMenu(session, range, "Clear active range");
  session.endZoom();
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, range, "Set as active range");
  await session.hold(1000);
  session.endZoom();

  await capture.openToolbar(session, "task");
  const task = page
    .locator(".console-task-toolbar")
    .getByText(TASK, { exact: true })
    .first();
  await session.waitFor(task);
  await session.hold(600);
  await capture.contextMenu(session, task, `Snapshot to ${RANGE}`);
  await session.hold(1000);
  session.endZoom();

  // Snapshots show only on the range overview.
  await capture.openToolbar(session, "range");
  await session.waitFor(range);
  await session.hold(400);
  await capture.contextMenu(session, range, "View details");
  session.endZoom();
  const snapshot = page
    .locator(".console-range__overview")
    .getByText(`${TASK} (Snapshot)`, { exact: true })
    .first();
  await session.waitFor(snapshot);
  await snapshot.scrollIntoViewIfNeeded();
  await session.hold(600);
  await session.moveTo(snapshot, { text: true });
  await session.hold(3000);
};
