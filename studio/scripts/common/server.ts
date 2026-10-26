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

import { DRIVER, fieldItem, recordTask, type TaskShot } from "./task";

/** A simulated server and how to connect it from the Console. */
export interface Server {
  /** The simulated server the Driver connects to. */
  server: fixtures.SimulatorOptions;
  /** The palette command that opens the connect dialog. */
  command: string;
  /** The name the dialog starts with, which is also its placeholder. */
  defaultName: string;
  /** The name to give the server. */
  name: string;
  /** Fills the fields of the protocol, under "Connect from". */
  fill: (session: capture.CaptureSession, modal: Locator) => Promise<void>;
}

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
 * browseNodes returns task channels that open MyObject in the OPC UA browser and drag
 * each of `nodes` from it into the channel list.
 */
export const browseNodes =
  (nodes: string[]): TaskShot["channels"] =>
  async (session, form) => {
    const browser = form.locator(".console-opc-browser").first();
    const node = (name: string): Locator =>
      browser.getByRole("treeitem").filter({ hasText: new RegExp(`^${name}$`) });
    await session.waitFor(node("MyObject"));
    await session.click(node("MyObject"), { text: true });
    const list = form.locator(".console-channel-list").first();
    for (const name of nodes) {
      await session.waitFor(node(name));
      await node(name).scrollIntoViewIfNeeded();
      await session.hold(400);
      await session.drag(node(name), list);
      await session.waitFor(list.getByRole("option").filter({ hasText: name }));
      await session.hold(500);
    }
  };

/**
 * connect runs the connect command of a server, enters its name and the Driver, fills
 * the fields of the protocol, and waits for the Driver to test the connection.
 */
const connect = async (
  session: capture.CaptureSession,
  { command, defaultName, name, fill }: Server,
): Promise<void> => {
  const { page } = session;
  await capture.commandPalette(session, command);
  const modal = page.locator(capture.MODAL).first();
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
 * serverTaskScript returns a script that connects a simulated server off camera and
 * records a task shot on it. The script module must export `driver = true`.
 */
export const serverTaskScript =
  (server: Server, shot: Omit<TaskShot, "device">) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const sim = await fixtures.simulator(server.server);
    try {
      await capture.login(session);
      await capture.clearPanel(session);
      await connect(session, server);
      await recordTask(session, { device: server.name, ...shot });
    } finally {
      await sim.stop();
    }
  };

/**
 * connectServerScript returns a script that connects a simulated server on camera. The
 * server then shows under the Driver in the devices toolbar. The script module must
 * export `driver = true`.
 */
export const connectServerScript =
  (server: Server) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const sim = await fixtures.simulator(server.server);
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
