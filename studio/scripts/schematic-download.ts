// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { Synnax } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

const NAME = "Feed System";
const DOWNLOADS = path.join(tmpdir(), "synnax-studio-schematic-download");

/**
 * Docs `console/schematics/download`: find the schematic under its project in
 * the Projects toolbar, open its context menu, and export it; the Console
 * streams the JSON to disk. The save itself stays off screen: the Console
 * reports it through the notification feed, which the rig hides.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const project = capture.PROJECT;
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
  await rename(await key(created), NAME);
  const item = capture.treeItem(page, "schematic:", NAME);
  await session.waitFor(item);

  await allowDownloads(session);
  await session.moveTo({ x: 900, y: 500 });

  session.startRecording();
  await session.hold(1200);

  const saved = saveDownload(session);
  // The save lands off screen, so the menu is the subject: dwell on it before
  // the click rather than holding on an unchanged tree after it.
  await session.rightClick(item, { zoom: false });
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await session.hold(1600);
  await session.click(menu.getByText("Export", { exact: true }).first(), {
    zoom: false,
  });
  await session.hold(1200);
  await saved;
};

/** key extracts a resource's key from its tree item's ontology-id DOM id. */
const key = async (item: Locator): Promise<string> => {
  const id = await item.getAttribute("id");
  if (id == null) throw new Error("tree item carries no id");
  return id.slice(id.indexOf(":") + 1);
};

const rename = async (schematic: string, name: string): Promise<void> => {
  const client = new Synnax({
    host: "localhost",
    port: fixtures.defaultPort(),
    username: "synnax",
    password: "seldon",
  });
  try {
    await client.schematics.rename(schematic, name);
  } finally {
    await client.close();
  }
};

/**
 * allowDownloads makes the Console's download land on the anchor path Playwright
 * handles. The Console prefers `showSaveFilePicker`, whose native dialog Playwright
 * cannot answer.
 */
const allowDownloads = async (session: capture.CaptureSession): Promise<void> => {
  await mkdir(DOWNLOADS, { recursive: true });
  await session.page.evaluate(() => {
    Object.defineProperty(window, "showSaveFilePicker", {
      value: undefined,
      configurable: true,
    });
  });
};

/** saveDownload writes the page's next download into this script's directory. */
const saveDownload = async (session: capture.CaptureSession): Promise<string> => {
  const download = await session.page.waitForEvent("download");
  const file = path.join(DOWNLOADS, download.suggestedFilename());
  await download.saveAs(file);
  return file;
};
