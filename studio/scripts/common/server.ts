// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { simulator, type SimulatorOptions } from "@/fixtures/simulator";
import { capture } from "@/index";

export const DRIVER = "Node 1 Embedded Driver";

/** A simulated server and how to connect it from the Console. */
export interface Server {
  /** The simulated server the Driver connects to. */
  server: SimulatorOptions;
  /** The palette command that opens the connect dialog. */
  command: string;
  /** The name the dialog starts with, which is also its placeholder. */
  defaultName: string;
  /** The name to give the server. */
  name: string;
  /** Fills the fields of the protocol, under "Connect from". */
  fill: (session: capture.CaptureSession, modal: Locator) => Promise<void>;
}

/** fieldItem returns the form field with the given label. */
const fieldItem = (modal: Locator, label: string): Locator =>
  modal.locator(".pluto-input__item").filter({ hasText: label }).first();

/** switchField returns the switch with the given label. */
const switchField = (modal: Locator, label: string): Locator =>
  fieldItem(modal, label).locator(".pluto-input__switch").first();

export const MODBUS: Server = {
  server: { protocol: "modbus", port: 5020 },
  command: "Connect Modbus server",
  defaultName: "Modbus server",
  name: "Pump skid PLC",
  fill: async (session, modal) => {
    await session.click(modal.getByPlaceholder("localhost").first());
    await session.type("127.0.0.1");
    await session.hold(300);
    await session.click(modal.getByPlaceholder("502").first());
    await session.press("ControlOrMeta+a");
    await session.type("5020");
  },
};

export const OPCUA: Server = {
  server: { protocol: "opcua", port: 4840 },
  command: "Connect OPC UA server",
  defaultName: "OPC UA server",
  name: "Furnace controller",
  fill: async (session, modal) => {
    await session.click(fieldItem(modal, "Endpoint").locator("input").first());
    await session.press("ControlOrMeta+a");
    await session.type("opc.tcp://127.0.0.1:4840");
  },
};

export const HTTP: Server = {
  server: { protocol: "http", port: 8080 },
  command: "Connect HTTP server",
  defaultName: "HTTP server",
  name: "Weather station",
  fill: async (session, modal) => {
    await session.click(modal.getByPlaceholder("www.example.com").first());
    await session.type("127.0.0.1:8080");
    await session.hold(300);
    await session.click(switchField(modal, "HTTPS"));
    await session.hold(400);
    await session.click(modal.getByPlaceholder("/health").first());
    await session.type("/health");
  },
};

/**
 * connect runs the connect command of a server, enters its name and the Driver, fills
 * the fields of the protocol, and waits for the Driver to test the connection.
 */
export const connect = async (
  session: capture.CaptureSession,
  { command, defaultName, name, fill }: Server,
): Promise<void> => {
  const { page } = session;
  await capture.commandPalette(session, command);
  const modal = page.locator(".pluto-dialog__dialog.pluto--modal").first();
  const nameInput = modal.getByPlaceholder(defaultName).first();
  await session.waitFor(nameInput);
  await session.zoom(modal);
  await session.hold(800);

  await session.click(nameInput, { text: true });
  await session.press("ControlOrMeta+a");
  await session.type(name);
  await session.hold(400);

  await session.click(
    fieldItem(modal, "Connect from").locator(".pluto-dialog__trigger").first(),
  );
  const rack = page.locator(".pluto-list__item").filter({ hasText: DRIVER }).first();
  await session.waitFor(rack);
  await session.hold(600);
  await session.click(rack, { text: true });
  await session.hold(400);

  await fill(session, modal);
  await session.hold(800);

  await capture.clickButton(session, "Connect");
  session.endZoom();
  await session.waitForHidden(modal);
};

/**
 * openTaskForm connects a server off camera and opens the devices toolbar on it, then
 * starts the recording, right-clicks the server, and selects `item`. It returns the
 * task form that opens.
 */
export const openTaskForm = async (
  session: capture.CaptureSession,
  server: Server,
  item: string,
): Promise<Locator> => {
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await connect(session, server);
  await capture.openToolbar(session, "device");
  await capture.resizeToolbar(session, 400);
  const driver = capture.treeItem(page, "rack:", DRIVER);
  await session.waitFor(driver);
  await session.click(driver, { text: true });
  const device = capture.treeItem(page, "device:", server.name);
  await session.waitFor(device);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await session.rightClick(device, { text: true });
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
  form: Locator,
  label: string,
  value: string,
): Promise<void> => {
  await session.click(fieldItem(form, label).locator("input").first());
  await session.press("ControlOrMeta+a");
  await session.type(value);
  await session.press("Enter");
};

/**
 * connectServerScript returns a script that connects a simulated server on camera. The
 * server then shows under the Driver in the devices toolbar. The script module must
 * export `driver = true`.
 */
export const connectServerScript =
  (server: Server) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const sim = await simulator(server.server);
    try {
      const { page } = session;
      await capture.login(session);
      await capture.clearPanel(session);
      await session.moveTo({ x: 800, y: 450 });

      session.startRecording();
      await session.hold(1500);

      await connect(session, server);
      await session.hold(600);

      await capture.openToolbar(session, "device");
      const driver = capture.treeItem(page, "rack:", DRIVER);
      await session.waitFor(driver);
      await session.click(driver, { text: true });
      await session.waitFor(capture.treeItem(page, "device:", server.name));
      await session.moveTo({ x: 1000, y: 500 });
      await session.hold(3000);
    } finally {
      await sim.stop();
    }
  };
