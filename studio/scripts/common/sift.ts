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

import { openProject } from "./desktop";
import { CORE_RACK, type Server } from "./server";

/** The port of the fake Sift server. */
const SIFT_PORT = 50551;

/** The URI of the fake Sift server, as a Sift connection gives it. */
const SIFT_URI = `localhost:${SIFT_PORT}`;

/** The name of the Sift connection that `siftScript` creates. */
export const CONNECTION = "Sift";

/** The Sift asset the videos send data to. */
export const ASSET = "TPC";

/** The API key the connect video types. The fake accepts every key. */
const API_KEY = "sk-7f3a9c41e2b84d6f";

/** The fake Sift server and how to connect it from the Console. */
export const SIFT: Server = {
  server: { protocol: "sift", port: SIFT_PORT },
  rack: CORE_RACK,
  command: "Connect Sift",
  title: "Sift",
  defaultName: "Sift",
  name: CONNECTION,
  fill: async (session, modal) => {
    const { page } = session;
    await session.click(modal.getByPlaceholder("grpc-api.siftstack.com:443").first());
    await session.type(SIFT_URI);
    await session.hold(400);
    await session.click(modal.getByText("Select a secret", { exact: true }).first(), {
      text: true,
    });
    const newSecret = page
      .locator(".lyra-select__dialog:visible")
      .getByText("New secret", { exact: true })
      .first();
    await session.waitFor(newSecret);
    await session.hold(400);
    await session.click(newSecret, { text: true });
    const secret = page.locator(capture.MODAL).last();
    const value = secret.getByPlaceholder("Value", { exact: true }).first();
    await session.waitFor(value);
    await session.hold(600);
    await session.click(value);
    await session.type(API_KEY);
    await session.hold(500);
    await session.click(secret.getByRole("button", { name: "Create", exact: true }));
    await session.waitFor(modal.getByText(`${CONNECTION} API key`, { exact: true }));
  },
};

/** createConnection creates a Sift connection to the fake on the rack of the Core. */
const createConnection = async (): Promise<void> => {
  const client = fixtures.connect();
  try {
    const [rack] = await client.racks.retrieve({ integrations: "sift" });
    const key = await client.secrets.set({
      source: "stored",
      name: "Sift API key",
      value: "studio",
    });
    await client.devices.create({
      key: "sift",
      name: CONNECTION,
      make: "sift",
      model: "sift",
      location: SIFT_URI,
      rack: rack.key,
      configured: true,
      properties: { uri: SIFT_URI, apiKey: key.key },
    });
  } finally {
    await client.close();
  }
};

/**
 * siftScript returns a script that runs the fake Sift server and the TPC world, creates
 * a Sift connection to the fake, opens the TPC project, and records `shot`. It stops
 * each fixture that started, also when a later one fails. The script module must export
 * `desktop = true`, so the video shows only the features of a release.
 */
export const siftScript =
  (shot: (session: capture.CaptureSession, world: fixtures.World) => Promise<void>) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const sift = await fixtures.simulator({ protocol: "sift", port: SIFT_PORT });
    try {
      const world = await fixtures.testStand({ now: () => session.now() });
      try {
        await createConnection();
        await openProject(session, world.project);
        await shot(session, world);
      } finally {
        await world.stop();
      }
    } finally {
      await sift.stop();
    }
  };

/**
 * fillUpload picks the asset and the channels in the open Upload to Sift dialog, then
 * clicks Upload. `search` is text that lists every one of the channels.
 */
export const fillUpload = async (
  session: capture.CaptureSession,
  modal: Locator,
  search: string,
  channels: string[],
): Promise<void> => {
  const { page } = session;
  // The modal zooms in as it opens, and the camera frames its box at the start.
  await session.settle(300);
  await session.zoom(modal);
  const asset = modal.getByText("Select asset", { exact: true }).first();
  await session.waitFor(asset);
  await session.click(asset, { text: true });
  await session.settleWall(1200);
  const tpc = page.getByRole("option", { name: ASSET, exact: true }).first();
  await session.waitFor(tpc);
  await session.click(tpc, { text: true });
  const select = modal.getByText("Select channels to upload").first();
  await session.click(select, { text: true });
  await session.hold(500);
  await session.type(search);
  for (const name of channels) {
    const option = page.getByRole("option", { name, exact: true }).first();
    await session.waitFor(option);
    await session.click(option, { text: true });
    await session.hold(250);
  }
  await page.keyboard.press("Escape");
  // The camera followed the clicks in the channel list, so frame the modal again.
  await session.zoom(modal);
  await session.hold(900);
  await session.click(modal.getByRole("button", { name: "Upload", exact: true }));
  session.endZoom();
};
