// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

/** Name of the embedded rack the Driver runs on in every capture's Core. */
export const DRIVER = "Node 1 Embedded Driver";

/** The devices the configure device videos leave behind. */
export const NI_DAQ = fixtures.configured(
  fixtures.NI_USB_6343,
  "Test stand DAQ",
  "dev1",
);
export const LABJACK = fixtures.configured(
  fixtures.LABJACK_T7,
  "Thrust stand T7",
  "lj1",
);
export const TANK_PRESSURES = fixtures.configured(
  fixtures.EL3102,
  "Tank pressures",
  "ai1",
);
export const VALVE_OUTPUTS = fixtures.configured(
  fixtures.EL2004,
  "Valve outputs",
  "do1",
);

/**
 * fieldItem returns the form field whose label is exactly `label`, with or without the
 * mark of a required field.
 */
export const fieldItem = (scope: Locator, label: string): Locator =>
  scope
    .locator(".lyra-input__item")
    .filter({
      has: scope
        .page()
        .locator(".lyra-input-label")
        .filter({ hasText: new RegExp(`^\\s*${label}\\s*\\*?\\s*$`) }),
    })
    .first();

/** setNumber replaces the value of the numeric field with the given label. */
const setNumber = async (
  session: capture.CaptureSession,
  scope: Locator,
  label: string,
  value: string,
): Promise<void> => {
  await session.click(fieldItem(scope, label).locator("input").first());
  await session.press("ControlOrMeta+a");
  await session.type(value);
  await session.press("Enter");
  await session.hold(300);
};

/**
 * choose opens the select with the given label and picks `item`. A long list renders
 * only its first items, so `search` types the item into the select's search first.
 */
export const choose = async (
  session: capture.CaptureSession,
  scope: Locator,
  label: string,
  item: string,
  { search = false }: { search?: boolean } = {},
): Promise<void> => {
  await session.click(fieldItem(scope, label).locator(".lyra-dialog__trigger").first());
  if (search) {
    await session.hold(400);
    await session.type(item);
  }
  const option = session.page
    .locator(".lyra-list__item:not(.lyra-tree__item)")
    .filter({ has: session.page.getByText(item, { exact: true }) })
    .first();
  await session.waitFor(option);
  await session.hold(500);
  await session.click(option.getByText(item, { exact: true }).first(), { text: true });
  await session.hold(400);
};

/** addChannel clicks "Add channel" in the form. */
export const addChannel = async (
  session: capture.CaptureSession,
  form: Locator,
): Promise<void> => {
  await session.click(form.getByRole("button", { name: "Add channel" }).first());
  await session.hold(600);
};

export interface TaskShot {
  /** The name of the device in the devices toolbar. */
  device: string;
  /** The context menu item that creates the task. */
  item: string;
  /** The value of each rate field, by label. */
  rates?: Record<string, string>;
  /** Adds and sets the channels of the task. */
  channels: (session: capture.CaptureSession, form: Locator) => Promise<void>;
}

/**
 * recordTask opens the devices toolbar on the Driver, starts the recording, and creates
 * a task from the device. It zooms on each area of work in turn: the device and its
 * context menu, the rates, and the channels. It then starts the task and holds the
 * whole Console on the status that the Driver reports. The device must be on the
 * Driver.
 */
export const recordTask = async (
  session: capture.CaptureSession,
  { device, item, rates = {}, channels }: TaskShot,
): Promise<void> => {
  const { page } = session;
  await capture.openToolbar(session, "device");
  await capture.resizeToolbar(session, 400);
  const driver = capture.treeItem(page, "rack:", DRIVER);
  await session.waitFor(driver);
  await session.click(driver, { text: true });
  const node = capture.treeItem(page, "device:", device);
  await session.waitFor(node);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, node, item);
  session.endZoom();
  const form = page.locator(".console-task-configure").first();
  await session.waitFor(form);
  await session.hold(1200);

  if (Object.keys(rates).length > 0) {
    await session.zoom(form.locator(".console-task-properties").first());
    await session.hold(400);
    for (const [label, value] of Object.entries(rates))
      await setNumber(session, form, label, value);
    await session.hold(300);
  }

  // New channels show at the top of the area and "Add channel" sits at its bottom, so
  // the zoom stays shallow enough to frame both.
  await capture.zoomCenter(
    session,
    form.locator(".console-task-channel-form-container"),
    1.35,
  );
  await session.hold(400);
  await channels(session, form);
  await session.hold(500);

  await session.click(form.locator("button:has(svg.lyra-icon--play)").first());
  session.endZoom();
  await session.waitFor(form.getByText("Started task").first());
  await session.moveTo({ x: 1100, y: 120 });
  await session.hold(3000);
};

/**
 * taskScript returns a script that records a task shot on a device of the simulated
 * Driver. Nothing reads or writes hardware, so the task produces no samples.
 */
export const taskScript =
  ({
    device,
    ...shot
  }: Omit<TaskShot, "device"> & { device: fixtures.SimulatedDevice }) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const driver = await fixtures.simulatedDriver({ devices: [device] });
    try {
      await capture.login(session);
      await capture.clearPanel(session);
      await recordTask(session, { device: device.name, ...shot });
    } finally {
      await driver.stop();
    }
  };
