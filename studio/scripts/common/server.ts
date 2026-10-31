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
  /** The entry of the server in the dialog that the devices toolbar opens. */
  title: string;
  /** The name the dialog starts with, which is also its placeholder. */
  defaultName: string;
  /** The name to give the server. */
  name: string;
  /** Fills the fields of the protocol. */
  fill: (session: capture.CaptureSession, modal: Locator) => Promise<void>;
}

/**
 * The integrations of the servers. A connect video enables only these, so the Driver
 * finds no hardware on the computer that records it.
 */
export const INTEGRATIONS = ["opc", "modbus", "http"];

/** checkboxField returns the checkbox with the given label. */
const checkboxField = (modal: Locator, label: string): Locator =>
  fieldItem(modal, label).getByRole("checkbox").first();

export const MODBUS: Server = {
  server: { protocol: "modbus", port: 5020 },
  command: "Connect Modbus server",
  title: "Modbus TCP server",
  defaultName: "Modbus server",
  name: "Pump skid PLC",
  fill: async (session, modal) => {
    await session.click(modal.getByPlaceholder("localhost").first());
    await session.type("127.0.0.1");
    await session.hold(300);
    await session.click(fieldItem(modal, "Port").locator("input").first());
    await session.press("ControlOrMeta+a");
    await session.type("5020");
  },
};

export const OPCUA: Server = {
  server: { protocol: "opcua", port: 4840 },
  command: "Connect OPC UA server",
  title: "OPC UA server",
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
  title: "HTTP server",
  defaultName: "HTTP server",
  name: "Weather station",
  fill: async (session, modal) => {
    await session.click(modal.getByPlaceholder("www.example.com").first());
    await session.type("127.0.0.1:8080");
    await session.hold(300);
    await session.click(checkboxField(modal, "HTTPS"));
    await session.hold(400);
    await session.click(modal.getByRole("tab", { name: "Health check" }), {
      text: true,
    });
    const path = modal.getByPlaceholder("/health").first();
    await session.waitFor(path);
    await session.click(path);
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
 * fillForm names the server in its open connect form, fills the fields of the protocol,
 * and waits for the Driver to test the connection. The Driver is the only rack, so the
 * form selects it.
 */
const fillForm = async (
  session: capture.CaptureSession,
  modal: Locator,
  { defaultName, name, fill }: Server,
): Promise<void> => {
  const nameInput = modal.getByPlaceholder(defaultName).first();
  await session.waitFor(nameInput);
  await session.zoom(modal);
  await session.hold(800);

  await session.click(nameInput, { text: true });
  await session.press("ControlOrMeta+a");
  await session.type(name);
  await session.hold(400);

  await fill(session, modal);
  await session.waitFor(fieldItem(modal, "Connect from").getByText(DRIVER));
  await session.hold(800);

  await capture.clickButton(session, "Connect");
  session.endZoom();
  await session.waitForHidden(modal);
};

/** connect connects a server through its palette command. */
const connect = async (
  session: capture.CaptureSession,
  server: Server,
): Promise<void> => {
  await capture.commandPalette(session, server.command);
  await fillForm(session, session.page.locator(capture.MODAL).first(), server);
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
 * connectServerScript returns a script that connects a simulated server on camera from
 * the devices toolbar. The server then shows under the Driver. The script module must
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

      await capture.openToolbar(session, "device");
      await session.hold(600);
      await capture.clickToolbarCreate(session);
      const modal = page.locator(capture.MODAL).first();
      const entry = modal.getByRole("button").filter({ hasText: server.title }).first();
      await session.waitFor(entry);
      await session.zoom(modal);
      await session.hold(1200);
      await session.click(entry, { text: true, zoom: false });
      await fillForm(session, modal, server);
      await session.hold(600);

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
