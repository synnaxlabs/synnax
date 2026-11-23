// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import type { Page } from "playwright";

import type { capture } from "@/index";

/** How far runAhead puts the virtual clock ahead of wall time. */
const LEAD = TimeSpan.minutes(30);

/**
 * runAhead runs the virtual clock ahead of wall time without capturing. The TPC stand
 * stamps samples on the later of the two clocks, and Arc runs on wall time. A capture
 * that stays ahead stamps its ranges on the clock the stand uses, and Arc sees each
 * sample as new. Call it before `fixtures.testStand`.
 */
export const runAhead = async (session: capture.CaptureSession): Promise<void> => {
  session.setSpeed(60);
  while (session.now().before(TimeStamp.now().add(LEAD))) await session.tick();
  session.setSpeed(1);
};

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
