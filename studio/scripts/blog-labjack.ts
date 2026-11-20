// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { choose, DRIVER, fieldItem } from "./common/task";

export const viewport = { width: 1280, height: 800 };

export const desktop = true;

export const driver = true;

/** Only LabJack, so the embedded Driver finds the T7 of the machine that captures. */
export const integrations = ["labjack"];

const NAME = "Lab T7";
const IDENTIFIER = "t7";
const CHANNEL = `${IDENTIFIER}_AIN0`;
/** How far above its steady reading the thermocouple reads once a hand warms it. */
const ONSET = 0.6;
/** How long the plot climbs after the onset, so its 30 s window shows the climb. */
const CLIMB_MS = 24_000;
/** How far apart, at most, the means of two windows of a steady thermocouple are. */
const STEADY = 0.05;
const STEADY_WINDOW_MS = 3_000;
const STEADY_TIMEOUT_MS = 90_000;
const WARM_TIMEOUT_MS = 180_000;
/** Set LABJACK_WARM=1 to wait for a person to warm the thermocouple. Drafts skip it. */
const WARM = process.env.LABJACK_WARM === "1";

/**
 * Stills for the blog post `labjack-t7-software`, from a real LabJack T7 on USB with a
 * K-type thermocouple on `AIN0` and `GND`. The script takes the post's steps in order:
 * it configures the T7, starts a read task with the thermocouple, plots it while a
 * person warms the thermocouple by hand, and selects the warm-up for export. Fails when
 * no T7 is on USB, or when the reading does not start to rise within three minutes.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    const { page } = session;
    await openProject(session);

    await capture.openToolbar(session, "device");
    await capture.resizeToolbar(session, 400);
    const rack = capture.treeItem(page, "rack:", DRIVER);
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    const found = capture.treeItem(page, "device:", "LJM_dtT7");
    await waitWall(session, found, 60_000);
    await capture.contextMenu(session, found, "Configure");
    const modal = page.locator(".console-configure").first();
    await session.waitFor(modal.locator("input").first());
    await session.click(modal.locator("input").first());
    await session.press("ControlOrMeta+a");
    await session.type(NAME);
    await session.click(fieldItem(modal, "Identifier").locator("input").first());
    await session.type(IDENTIFIER);
    await session.moveTo(capture.PARK);
    await session.settle(600);
    await session.still("configure", { dialog: modal });
    await capture.clickButton(session, "Save and create read task");
    await session.waitForHidden(modal);
    await capture.closeToolbar(session);

    const form = page.locator(".console-task-configure").first();
    await session.waitFor(form);
    // A new read task starts with one analog channel on AIN0.
    await session.click(form.getByText("AIN0", { exact: true }).first(), {
      text: true,
    });
    await choose(session, form, "Channel type", "Thermocouple");
    await fieldItem(form, "Temperature units").scrollIntoViewIfNeeded();
    await choose(session, form, "Temperature units", "Celsius");
    await session.click(form.locator("button:has(svg.lyra-icon--play)").first());
    const started = form.getByText("Started task").first();
    await waitWall(session, started, 30_000);
    const units = fieldItem(form, "Temperature units");
    await units.scrollIntoViewIfNeeded();
    await session.moveTo(capture.PARK);
    await session.settleWall(2000);
    await session.settle(600);
    await session.still("task", {
      settings: [
        form.getByText("Thermocouple type", { exact: true }),
        form.locator("button:has(svg.lyra-icon--stop)"),
        started,
      ],
    });

    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Line plot");
    const plot = page.locator(".pluto-line-plot").first();
    await session.waitFor(plot);
    await capture.addChannels(session, "Y1", [CHANNEL], { search: CHANNEL });
    await capture.hideBottomToolbar(session);
    await session.moveTo(capture.PARK);

    await warmUp(client, session);
    await session.settle(600);
    await session.still("window");

    await selectWarmUp(session, plot);
    await session.still("export", { menu: page.locator(".lyra-menu-context").first() });
    if (WARM) console.log(">>> Let go of the thermocouple.");
  } finally {
    await client.close();
  }
};

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

/** openProject creates an empty project from the projects screen of Synnax Desktop. */
const openProject = async (session: capture.CaptureSession): Promise<void> => {
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
 * warmUp waits for the thermocouple to read steady, asks the person at the T7 to warm
 * it, and returns CLIMB_MS after the reading starts to rise.
 */
const warmUp = async (
  client: ReturnType<typeof fixtures.connect>,
  session: capture.CaptureSession,
): Promise<void> => {
  if (!WARM) {
    await session.settleWall(20000);
    return;
  }
  const { key } = await client.channels.retrieve(CHANNEL);
  const streamer = await client.openStreamer([key]);
  const samples: { at: number; value: number }[] = [];
  const loop = (async () => {
    for await (const frame of streamer)
      for (const sample of frame.get(key))
        samples.push({ at: Date.now(), value: Number(sample) });
  })();
  /** The mean of the samples from `from` ms ago to `to` ms ago, or NaN for none. */
  const mean = (from: number, to: number = 0): number => {
    const now = Date.now();
    const values = samples
      .filter(({ at }) => at > now - from && at <= now - to)
      .map(({ value }) => value);
    return values.reduce((sum, v) => sum + v, 0) / values.length;
  };
  let report = Date.now();
  const wait = async (until: () => boolean, ms: number): Promise<boolean> => {
    const deadline = Date.now() + ms;
    while (!until()) {
      if (Date.now() > deadline) return false;
      if (Date.now() >= report) {
        console.log(`${CHANNEL}: ${mean(1000).toFixed(2)} °C`);
        report += 5000;
      }
      await session.settleWall(250);
    }
    return true;
  };
  // The client logs out on close and ends any open stream, so close the stream first.
  try {
    // A thermocouple still warm from the last theme's capture would climb downhill.
    await wait(
      () =>
        Math.abs(
          mean(STEADY_WINDOW_MS) - mean(2 * STEADY_WINDOW_MS, STEADY_WINDOW_MS),
        ) < STEADY,
      STEADY_TIMEOUT_MS,
    );
    const steady = mean(STEADY_WINDOW_MS);
    console.log(`>>> Hold the thermocouple in your hand now (${CHANNEL}).`);
    if (!(await wait(() => mean(1000) - steady >= ONSET, WARM_TIMEOUT_MS)))
      throw new Error(
        `${CHANNEL} did not rise ${ONSET} °C above ${steady.toFixed(2)} °C within ` +
          "three minutes",
      );
    await wait(() => false, CLIMB_MS);
  } finally {
    streamer.close();
    await loop;
  }
};

/** selectWarmUp holds Alt, drags across the climb, and opens the selection's menu. */
const selectWarmUp = async (
  session: capture.CaptureSession,
  plot: Locator,
): Promise<void> => {
  const box = await plot.boundingBox();
  if (box == null) throw new Error("the line plot is hidden");
  const at = (fx: number, fy: number): { x: number; y: number } => ({
    x: box.x + box.width * fx,
    y: box.y + box.height * fy,
  });
  const { page } = session;
  await page.keyboard.down("Alt");
  await session.drag(at(0.35, 0.2), at(0.85, 0.8));
  await page.keyboard.up("Alt");
  await session.rightClick(at(0.6, 0.5));
  await session.waitFor(page.locator(".lyra-menu-context").first());
  await session.settle(600);
};
