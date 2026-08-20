// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { imex, schematic, Synnax } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

const NAME = "Feed System";
const FIXTURES = path.join(tmpdir(), "synnax-studio-schematic-upload");

/**
 * Docs `console/schematics/upload`: right-click a project in the Projects
 * toolbar, import a schematic JSON from disk, and the schematic opens as a new
 * tab under the project.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.place(session, "Tank", { x: 480, y: 240 });
  await capture.place(session, "Gate", { x: 900, y: 250 });
  await capture.deselect(session, { x: 1300, y: 520 });
  // Close the symbol library so the canvas fills the panel.
  await page.locator(".console-main-nav__item").last().click();
  await session.settle(400);

  await capture.openToolbar(session, "project");
  await capture.resizeToolbar(session, 380);
  const projectItem = capture.treeItem(page, "project:", project);
  await session.waitFor(projectItem);
  await projectItem.click();
  const created = capture.treeItem(page, "schematic:", "Schematic");
  await session.waitFor(created);
  // The fixture is a real export of a Console schematic. Its source is removed so
  // the import is the only one on the cluster.
  const file = await writeFixture(await key(created), NAME);
  await session.waitForHidden(created);
  await capture.clearPanel(session);
  await session.moveTo({ x: 900, y: 500 });

  session.startRecording();
  await session.hold(1200);

  const chooser = page.waitForEvent("filechooser");
  await capture.contextMenu(session, projectItem, "Import components");
  await (await chooser).setFiles(file);

  const imported = capture.treeItem(page, "schematic:", NAME);
  await session.waitFor(imported);
  await session.waitFor(page.locator(".pluto-diagram").first());
  await session.hold(2800);
};

/** key extracts a resource's key from its tree item's ontology-id DOM id. */
const key = async (item: Locator): Promise<string> => {
  const id = await item.getAttribute("id");
  if (id == null) throw new Error("tree item carries no id");
  return id.slice(id.indexOf(":") + 1);
};

/**
 * writeFixture renames the schematic, exports it to a JSON file this script owns,
 * then deletes it from the cluster. Returns the file's path.
 */
const writeFixture = async (schematicKey: string, name: string): Promise<string> => {
  const client = new Synnax({
    host: "localhost",
    port: fixtures.defaultPort(),
    username: "synnax",
    password: "seldon",
  });
  try {
    await client.schematics.rename(schematicKey, name);
    const stream = await client.imex.export(
      schematic.ontologyID(schematicKey),
      imex.JSON_OPTIONS,
    );
    await mkdir(FIXTURES, { recursive: true });
    const file = path.join(FIXTURES, `${name}.json`);
    await writeFile(file, await new Response(stream).text());
    await client.schematics.delete(schematicKey);
    return file;
  } finally {
    await client.close();
  }
};
