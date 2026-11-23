// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Synnax } from "@synnaxlabs/client";
import type { Locator } from "playwright";

import { capture } from "@/index";

import { enterIdentifier } from "./configure-device";
import { choose, DRIVER, fieldItem } from "./task";

// The LabJack blog post captures run Synnax Desktop against a real LabJack T7 on USB,
// with a type K thermocouple on AIN0 and GND.

export const desktop = true;

export const driver = true;

/**
 * The integrations of Synnax Desktop on macOS that need no hardware, so the connect
 * dialog shows no warnings. The Driver finds the T7 of the machine that captures.
 */
export const integrations = ["labjack", "opc", "modbus", "http"];

const NAME = "Lab T7";
const IDENTIFIER = "t7";
const CHANNEL = `${IDENTIFIER}_AIN0`;

/** How far above its steady reading the thermocouple reads once a hand warms it. */
const ONSET = 0.6;
/** How far apart, at most, the means of two windows of a steady thermocouple are. */
const STEADY = 0.05;
const STEADY_WINDOW_MS = 3_000;
const STEADY_TIMEOUT_MS = 90_000;
const WARM_TIMEOUT_MS = 180_000;
/** Set LABJACK_WARM=1 to wait for a person to warm the thermocouple. Drafts skip it. */
const WARM = process.env.LABJACK_WARM === "1";

/**
 * waitWall waits in wall time for the locator to show, since the T7 and the Driver run
 * on wall time that the capture's ticks outrun. Throws after `ms`.
 */
const waitWall = async (
  session: capture.CaptureSession,
  locator: Locator,
  ms: number,
): Promise<void> => {
  const deadline = Date.now() + ms;
  while (!(await locator.isVisible().catch(() => false))) {
    // One last tick-based wait saves the rig's timeout screenshot and throws.
    if (Date.now() > deadline) return await session.waitFor(locator, 1);
    await session.settleWall(250);
  }
};

/**
 * waitWallGone waits in wall time for the locator to hide. It records no frames, so
 * call it only while the screen holds still. Throws after `ms`.
 */
export const waitWallGone = async (
  session: capture.CaptureSession,
  locator: Locator,
  ms: number,
): Promise<void> => {
  const deadline = Date.now() + ms;
  while (await locator.isVisible().catch(() => false)) {
    if (Date.now() > deadline) throw new Error(`still visible after ${ms} ms`);
    await session.settleWall(250);
  }
};

/** openProject creates an empty project from the projects screen of Synnax Desktop. */
export const openProject = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const create = page.getByText("New project").first();
  await session.waitFor(create, 1800);
  await create.click();
  const name = page.locator(capture.MODAL).getByPlaceholder("Name").first();
  await session.waitFor(name);
  await name.fill("Thermocouple test");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await session.waitFor(page.locator(".console-palette button"), 1800);
};

/**
 * findT7 opens the devices toolbar and waits for the Driver to find the T7 under its
 * rack. Throws when no T7 shows within a minute.
 */
export const findT7 = async (session: capture.CaptureSession): Promise<Locator> => {
  const { page } = session;
  await capture.openToolbar(session, "device");
  await capture.resizeToolbar(session, 400);
  const rack = capture.treeItem(page, "rack:", DRIVER);
  await session.waitFor(rack);
  await session.click(rack, { text: true });
  const found = capture.treeItem(page, "device:", "LJM_dtT7");
  await waitWall(session, found, 60_000);
  return found;
};

/**
 * configure enters the name and identifier in the open configure dialog, then saves and
 * opens a read task. Returns the read task form.
 */
export const configure = async (
  session: capture.CaptureSession,
  dialog: Locator,
): Promise<Locator> => {
  const { page } = session;
  const name = dialog.locator("input").first();
  await session.waitFor(name);
  await session.click(name, { text: true });
  await session.press("ControlOrMeta+a");
  await session.type(NAME);
  await session.hold(400);
  await enterIdentifier(session, dialog, IDENTIFIER);
  await session.hold(600);
  await capture.clickButton(session, "Save and create read task");
  const form = page.locator(".console-task-configure").first();
  await session.waitFor(form);
  return form;
};

/** openTask finds and configures the T7, and returns its new read task form. */
export const openTask = async (session: capture.CaptureSession): Promise<Locator> => {
  const found = await findT7(session);
  await capture.contextMenu(session, found, "Configure");
  session.endZoom();
  const form = await configure(
    session,
    session.page.locator(".console-configure").first(),
  );
  await capture.closeToolbar(session);
  return form;
};

/**
 * readThermocouple makes the task's AIN0 channel a thermocouple in Celsius and starts
 * the task. Returns the "Started task" status.
 */
export const readThermocouple = async (
  session: capture.CaptureSession,
  form: Locator,
): Promise<Locator> => {
  // A new read task starts with its one channel, on AIN0, already selected.
  await choose(session, form, "Channel type", "Thermocouple");
  await fieldItem(form, "Temperature units").scrollIntoViewIfNeeded();
  await choose(session, form, "Temperature units", "Celsius");
  await session.click(form.locator("button:has(svg.lyra-icon--play)").first());
  const started = form.getByText("Started task").first();
  await waitWall(session, started, 30_000);
  return started;
};

/** The "Plot channels" button at the top right of a task form. */
export const plotButton = (form: Locator): Locator =>
  form.locator("button:has(svg.lyra-icon--line-plot)").first();

/** A watch on the thermocouple, for a capture that needs a person to warm it. */
export interface Thermocouple {
  /** ask tells the person at the T7 to hold the thermocouple. */
  ask: () => void;
  /**
   * rise calls `step` until the reading rises above its steady reading. Pass a wall
   * wait before recording, and `session.hold` to record the wait. Throws after three
   * minutes.
   */
  rise: (step: () => Promise<void>) => Promise<void>;
  /** close ends the stream. Call it before the client closes. */
  close: () => Promise<void>;
}

/**
 * watch streams the thermocouple and waits in wall time for it to read steady, so a
 * thermocouple still warm from the last capture does not climb downhill. Without
 * LABJACK_WARM=1 it only lets the plot buffer fill, and `rise` returns at once.
 */
export const watch = async (
  client: Synnax,
  session: capture.CaptureSession,
): Promise<Thermocouple> => {
  if (!WARM) {
    await session.settleWall(5000);
    return { ask: () => {}, rise: async () => {}, close: async () => {} };
  }
  const { key } = await client.channels.retrieve(CHANNEL);
  const streamer = await client.openStreamer([key]);
  const samples: { at: number; value: number }[] = [];
  const loop = (async () => {
    for await (const frame of streamer)
      for (const sample of frame.get(key))
        samples.push({ at: Date.now(), value: Number(sample) });
  })();
  const close = async (): Promise<void> => {
    streamer.close();
    await loop;
  };
  /** The mean of the samples from `from` ms ago to `to` ms ago, or NaN for none. */
  const mean = (from: number, to: number = 0): number => {
    const now = Date.now();
    const values = samples
      .filter(({ at }) => at > now - from && at <= now - to)
      .map(({ value }) => value);
    return values.reduce((sum, v) => sum + v, 0) / values.length;
  };
  let report = Date.now();
  const log = (): void => {
    if (Date.now() < report) return;
    console.log(`${CHANNEL}: ${mean(1000).toFixed(2)} °C`);
    report += 5000;
  };
  // A failed wait leaves no caller to close the stream, so close it here.
  // A window with no samples yet has a NaN mean, which must never pass a check.
  let steady = NaN;
  try {
    const deadline = Date.now() + STEADY_TIMEOUT_MS;
    while (
      !(
        Math.abs(
          mean(STEADY_WINDOW_MS) - mean(2 * STEADY_WINDOW_MS, STEADY_WINDOW_MS),
        ) < STEADY
      ) &&
      Date.now() < deadline
    ) {
      log();
      await session.settleWall(250);
    }
    steady = mean(STEADY_WINDOW_MS);
    if (Number.isNaN(steady))
      throw new Error(`${CHANNEL} streamed no samples in ${STEADY_TIMEOUT_MS} ms`);
  } finally {
    if (Number.isNaN(steady)) await close();
  }
  return {
    ask: () => console.log(`>>> Hold the thermocouple in your hand now (${CHANNEL}).`),
    rise: async (step) => {
      const deadline = Date.now() + WARM_TIMEOUT_MS;
      while (!(mean(1000) - steady >= ONSET)) {
        if (Date.now() > deadline)
          throw new Error(
            `${CHANNEL} did not rise ${ONSET} °C above ${steady.toFixed(2)} °C ` +
              "within three minutes",
          );
        log();
        await step();
      }
    },
    close,
  };
};

/**
 * warmUp asks the person at the T7 to warm the thermocouple, and returns once the
 * reading starts to rise. It waits in wall time, so it runs before recording.
 */
export const warmUp = async (
  client: Synnax,
  session: capture.CaptureSession,
): Promise<void> => {
  const thermocouple = await watch(client, session);
  // The client logs out on close and ends any open stream, so close the stream first.
  try {
    thermocouple.ask();
    await thermocouple.rise(() => session.settleWall(250));
  } finally {
    await thermocouple.close();
  }
};

/** letGo tells the person at the T7 that the capture no longer needs the warm-up. */
export const letGo = (): void => {
  if (WARM) console.log(">>> Let go of the thermocouple.");
};

/** at returns the point at fractions of the plot's box. */
export const at = async (
  plot: Locator,
  fx: number,
  fy: number,
): Promise<{ x: number; y: number }> => {
  const box = await plot.boundingBox();
  if (box == null) throw new Error("the line plot is hidden");
  return { x: box.x + box.width * fx, y: box.y + box.height * fy };
};
