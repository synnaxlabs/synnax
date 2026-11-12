// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { createKey, openCredentials } from "./common/credentials";
import { DRIVER, rackItem } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/enterprise/driver/install` (Log in a Driver with an API key): in the
 * devices toolbar, right-click the rack, select Manage sign-in, and create a key for
 * the Driver.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await capture.openToolbar(session, "device");
  await capture.resizeToolbar(session, 400);
  const rack = rackItem(page, DRIVER);
  await session.waitFor(rack);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1800);

  const modal = await openCredentials(session, rack);
  await createKey(session, modal, "Test stand Driver");
  session.endZoom();
  await session.hold(2000);
};
