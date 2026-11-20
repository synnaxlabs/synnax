// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { INTEGRATIONS, OPCUA } from "./common/server";
import { DRIVER, fieldItem, rackItem, setText } from "./common/task";

export const driver = true;
export const integrations = INTEGRATIONS;

/**
 * Docs `reference/hardware/opc-ua/connect-server` (Server certificate): connect a
 * simulated server that requires encryption. The first Connect fails with "Server
 * certificate not trusted" and shows the certificate that the server sent. Trust and
 * connect saves the certificate and connects again.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const sim = await fixtures.simulator({ protocol: "opcua-encrypted", port: 4840 });
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
    const entry = modal.getByRole("button").filter({ hasText: OPCUA.title }).first();
    await session.waitFor(entry);
    await session.zoom(modal);
    await session.hold(1000);
    await session.click(entry, { text: true, zoom: false });

    const name = modal.getByPlaceholder(OPCUA.defaultName).first();
    await session.waitFor(name);
    await session.hold(600);
    await session.click(name, { text: true });
    await session.press("ControlOrMeta+a");
    await session.type("Furnace controller");
    await session.hold(300);
    await setText(session, modal, "Endpoint", "opc.tcp://127.0.0.1:4840");
    await session.click(
      fieldItem(modal, "Mode").getByText("Sign and encrypt", { exact: true }),
      { text: true },
    );
    await session.hold(600);
    await session.waitFor(fieldItem(modal, "Connect from").getByText(DRIVER));
    await session.hold(800);

    await capture.clickButton(session, "Connect");
    const trust = modal.getByRole("button", { name: "Trust and connect", exact: true });
    await session.waitFor(trust, 1800);
    await trust.scrollIntoViewIfNeeded();
    await session.hold(2500);
    await session.click(trust);
    session.endZoom();
    await session.waitForHidden(modal, 1800);
    const rack = rackItem(page, DRIVER);
    await session.waitFor(rack);
    await session.click(rack, { text: true });
    await session.waitFor(capture.treeItem(page, "device:", "Furnace controller"));
    await session.moveTo({ x: 1000, y: 500 });
    await session.hold(3000);
  } finally {
    await sim.stop();
  }
};
