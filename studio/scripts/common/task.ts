// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import {
  configured,
  EL2004,
  EL3102,
  LABJACK_T7,
  NI_USB_6343,
  type SimulatedDevice,
  simulatedDriver,
} from "@/fixtures/hardware";
import { capture } from "@/index";

export const DRIVER = "Node 1 Embedded Driver";

/** The devices the configure device videos leave behind. */
export const NI_DAQ = configured(NI_USB_6343, "Test stand DAQ", "dev1");
export const LABJACK = configured(LABJACK_T7, "Thrust stand T7", "lj1");
export const TANK_PRESSURES = configured(EL3102, "Tank pressures", "ai1");
export const VALVE_OUTPUTS = configured(EL2004, "Valve outputs", "do1");

/**
 * fieldItem returns the form field whose label is exactly `label`, with or without
 * the mark of a required field.
 */
export const fieldItem = (scope: Locator, label: string): Locator =>
  scope
    .locator(".pluto-input__item")
    .filter({
      has: scope
        .page()
        .locator(".pluto-input-label")
        .filter({ hasText: new RegExp(`^\\s*${label}\\s*\\*?\\s*$`) }),
    })
    .first();

/**
 * openTaskForm opens the devices toolbar on the Driver off camera, then starts the
 * recording, right-clicks the device, and selects `item`. It returns the task form that
 * opens.
 */
export const openTaskForm = async (
  session: capture.CaptureSession,
  device: string,
  item: string,
): Promise<Locator> => {
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

  await session.rightClick(node, { text: true });
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await session.hold(500);
  await session.click(menu.getByText(item, { exact: true }).first(), { text: true });
  const form = page.locator(".console-task-configure").first();
  await session.waitFor(form);
  await session.hold(1000);
  return form;
};

/** setNumber replaces the value of the numeric field with the given label. */
export const setNumber = async (
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
  await session.click(
    fieldItem(scope, label).locator(".pluto-dialog__trigger").first(),
  );
  if (search) {
    await session.hold(400);
    await session.type(item);
  }
  const option = session.page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
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

/**
 * startTask clicks Start, ends the zoom, and holds on the status that the Driver
 * reports.
 */
export const startTask = async (
  session: capture.CaptureSession,
  form: Locator,
): Promise<void> => {
  await session.hold(500);
  await session.click(form.locator("button:has(svg.pluto-icon--play)").first());
  session.endZoom();
  await session.waitFor(form.getByText("Task started successfully").first());
  await session.moveTo({ x: 1100, y: 120 });
  await session.hold(3000);
};

export interface TaskShot {
  /** A configured device on the simulated Driver. */
  device: SimulatedDevice;
  /** The context menu item that creates the task. */
  item: string;
  /** Sets the fields of the task and adds its channels, under the zoom. */
  fill: (session: capture.CaptureSession, form: Locator) => Promise<void>;
}

/**
 * taskScript returns a script that creates a task from a device on the simulated
 * Driver, fills it, and starts it. Nothing reads or writes hardware, so the task
 * produces no samples.
 */
export const taskScript =
  ({ device, item, fill }: TaskShot) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const driver = await simulatedDriver({ devices: [device] });
    try {
      await capture.login(session);
      await capture.clearPanel(session);
      const form = await openTaskForm(session, device.name, item);
      await session.zoom(form);
      await fill(session, form);
      await startTask(session, form);
    } finally {
      await driver.stop();
    }
  };
