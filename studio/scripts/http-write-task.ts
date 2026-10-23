// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { simulator } from "@/fixtures/simulator";
import { type capture } from "@/index";

import { HTTP, openTaskForm } from "./common/server";

export const driver = true;

/**
 * Docs `reference/hardware/http/write-task` (Create an HTTP write task): create a write
 * task from a simulated HTTP server, add a PUT endpoint that sends the command value as
 * `/value`, and start the task.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const sim = await simulator(HTTP.server);
  try {
    const form = await openTaskForm(session, HTTP, "Create write task");

    await session.zoom(form);
    await session.click(form.getByText("Add endpoint", { exact: true }).first(), {
      text: true,
    });
    const path = form.getByPlaceholder("/api/control").first();
    await session.waitFor(path);
    await session.hold(500);
    await session.click(form.getByText("PUT", { exact: true }).first(), { text: true });
    await session.hold(400);
    await session.click(path);
    await session.type("/api/v1/setpoint");
    await session.hold(400);
    await session.click(form.getByPlaceholder("/value").first());
    await session.type("/value");
    await session.hold(800);

    await session.click(form.locator("button:has(svg.pluto-icon--play)").first());
    session.endZoom();
    await session.waitFor(form.getByText("Task started successfully").first());
    await session.moveTo({ x: 1100, y: 120 });
    await session.hold(3000);
  } finally {
    await sim.stop();
  }
};
