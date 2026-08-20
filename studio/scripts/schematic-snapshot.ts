// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

const NAME = "Feed System";
const RANGE = "Hotfire 09";

/**
 * Docs `console/schematics/snapshot`: right-click the schematic in the Projects
 * toolbar and snapshot it to the active range; the snapshot joins the range's
 * overview.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedRanges([RANGE]);
  const { page } = session;
  const project = capture.PROJECT;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.place(session, "Tank", { x: 480, y: 240 });
  await capture.place(session, "Gate", { x: 900, y: 250 });
  await capture.deselect(session, { x: 1300, y: 520 });
  // Close the symbol library so the schematic tab closes cleanly.
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
  await capture.clearPanel(session);

  // The snapshot lands on the active range, and its overview is where it shows up.
  await capture.searchPalette(session, RANGE);
  await capture.openToolbar(session, "range");
  const drawer = page.locator(".console-nav__drawer").first();
  await drawer.getByText(RANGE, { exact: true }).first().click();
  await session.waitFor(
    drawer.locator(".console-range-list-item.pluto--selected").first(),
  );
  // Switching toolbars remounts the tree, so the project collapses again.
  await capture.openToolbar(session, "project");
  await projectItem.click();
  await session.waitFor(item);
  const snapshots = page.getByText("Snapshots", { exact: true }).first();
  await session.waitFor(snapshots);
  await snapshots.scrollIntoViewIfNeeded();
  await session.settle(400);
  await session.moveTo({ x: 900, y: 500 });

  session.startRecording();
  await session.hold(1200);

  await capture.contextMenu(session, item, `Snapshot to ${RANGE}`);
  await session.waitFor(
    page.locator(".console-snapshots__list-item").filter({ hasText: NAME }).first(),
  );
  await session.hold(2800);
};

/** key extracts a resource's key from its tree item's ontology-id DOM id. */
const key = async (item: Locator): Promise<string> => {
  const id = await item.getAttribute("id");
  if (id == null) throw new Error("tree item carries no id");
  return id.slice(id.indexOf(":") + 1);
};

const rename = async (schematicKey: string, name: string): Promise<void> => {
  const client = new Synnax({
    host: "localhost",
    port: fixtures.defaultPort(),
    username: "synnax",
    password: "seldon",
  });
  try {
    await client.schematics.rename(schematicKey, name);
  } finally {
    await client.close();
  }
};
