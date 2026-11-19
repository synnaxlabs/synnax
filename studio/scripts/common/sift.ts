// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type capture, fixtures } from "@/index";

import { openProject } from "./desktop";

/** The port of the fake Sift server. */
export const SIFT_PORT = 50551;

/** The URI of the fake Sift server, as a Sift connection gives it. */
export const SIFT_URI = `localhost:${SIFT_PORT}`;

/** The name of the Sift connection that `siftScript` creates. */
export const CONNECTION = "Sift";

/** The Sift asset the videos send data to. */
export const ASSET = "TPC";

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
