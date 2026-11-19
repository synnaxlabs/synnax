// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Page } from "playwright";

import { type capture } from "@/index";

/** pinClock pauses the page's clock at the virtual time, so only ticks move it. */
export const pinClock = async (session: capture.CaptureSession): Promise<void> => {
  const { clock } = session.page;
  await clock.pauseAt((await session.page.evaluate(() => Date.now())) + 1000);
  await clock.setSystemTime(session.now().date());
};

/**
 * dropTemporal makes the page and its workers read time from `Date`, which the rig
 * steps, instead of `Temporal`, which it does not.
 */
export const dropTemporal = async (page: Page): Promise<void> => {
  const drop = (): void => {
    Reflect.deleteProperty(globalThis, "Temporal");
  };
  await page.evaluate(drop);
  for (const worker of page.workers()) await worker.evaluate(drop);
};
