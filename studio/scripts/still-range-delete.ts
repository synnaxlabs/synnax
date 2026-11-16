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

export const viewport = { width: 1280, height: 800 };

const RANGE = "Hotfire 05";

/**
 * Still for docs image `client/ranges/delete`: the dialog that confirms the delete of a
 * range, open from the context menu of the range explorer.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const now = TimeStamp.now();
  await fixtures.ensureRange({
    name: RANGE,
    timeRange: new TimeRange(now.sub(TimeSpan.hours(1)), now.add(TimeSpan.hours(1))),
  });
  await capture.login(session);
  await capture.openToolbar(session, "range");
  await page.getByText("Open range explorer", { exact: true }).first().click();
  const item = page.getByRole("listitem", { name: RANGE, exact: true }).first();
  await session.waitFor(item);
  await capture.closeToolbar(session);
  await capture.contextMenu(session, item, "Delete");
  session.endZoom();
  await session.waitFor(page.locator(capture.MODAL).first());
  await session.moveTo({ x: 26, y: 600 });
  await session.settleWall(800);
  await session.settle(1000);
  await session.still("window");
};
