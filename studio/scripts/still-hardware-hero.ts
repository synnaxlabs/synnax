// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, project, type Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { DRIVER, LABJACK } from "./common/task";

export const viewport = { width: 1280, height: 800 };

/** The stand's rack, which runs its DAQ task. */
const RACK = "Test stand DAQ";

/**
 * Fractions of the panel that put the right and bottom edges of the form's leaf just
 * past the part of the window the hero shows, so the hero hides the other leaves.
 */
const FORM_WIDTH = 0.755;
const FORM_HEIGHT = 0.908;

const retrievePanel = async (client: Synnax, name: string): Promise<panel.Panel> => {
  const found = (await client.panels.retrieve({ searchTerm: name })).find(
    (p) => p.name === name,
  );
  if (found == null) throw new Error(`no panel named "${name}"`);
  return found;
};

const tabs = (node: panel.Node): panel.Tab[] =>
  node.variant === "leaf" ? node.tabs : [...tabs(node.first), ...tabs(node.last)];

/**
 * Still for the hero of docs page `reference/hardware`: the devices toolbar with the
 * Drivers of the TPC stand and their devices, beside the running NI analog read task
 * that acquires the stand's sensors. The devices are simulated.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const driver = await fixtures.simulatedDriver({
    devices: [LABJACK, fixtures.MODBUS_PLC],
  });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const tpc = (await client.projects.retrieve({ searchTerm: world.project })).find(
      (p) => p.name === world.project,
    );
    if (tpc == null) throw new Error(`no project named "${world.project}"`);
    const hardware = await retrievePanel(client, "Hardware");
    const operations = tabs((await retrievePanel(client, "Operations")).root);
    const leaf = (type: string): panel.LeafNode => {
      const tab = operations.find(
        (t) => t.variant === "resource" && t.resource.type === type,
      );
      if (tab == null) throw new Error(`the Operations panel shows no ${type}`);
      return { variant: "leaf", tabs: [{ ...tab, key: crypto.randomUUID() }] };
    };
    await client.panels.delete(hardware.key);
    await client.panels.create({
      name: hardware.name,
      parent: project.ontologyID(tpc.key),
      root: {
        variant: "split",
        direction: "x",
        size: FORM_WIDTH,
        first: {
          variant: "split",
          direction: "y",
          size: FORM_HEIGHT,
          first: hardware.root,
          last: leaf("log"),
        },
        last: leaf("lineplot"),
      },
    });

    await capture.login(session, world.project);
    await session.click(page.getByText("Hardware", { exact: true }).first());
    await session.waitFor(page.locator(".console-task-configure:visible").first());
    await capture.openToolbar(session, "device");
    // A row click also selects the row, and the tree always keeps one row selected, so
    // the stand's rack goes last. Dispatched clicks leave no row in its hover state.
    for (const rack of [DRIVER, RACK]) {
      const item = capture.treeItem(page, "rack:", rack);
      await session.waitFor(item);
      await item.dispatchEvent("click");
      await session.settle(400);
    }
    await session.waitFor(capture.treeItem(page, "device:", "NI PXIe-6363"));
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await driver.stop();
    await world.stop();
  }
};
