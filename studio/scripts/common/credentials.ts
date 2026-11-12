// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture } from "@/index";

/** openCredentials opens the sign-in dialog of `item` and zooms on it. */
export const openCredentials = async (
  session: capture.CaptureSession,
  item: Locator,
): Promise<Locator> => {
  const { page } = session;
  await capture.contextMenu(session, item, "Manage sign-in");
  const modal = page.locator(capture.MODAL).first();
  await session.waitFor(modal.getByPlaceholder("Key name", { exact: true }));
  await session.hold(600);
  await session.zoom(modal);
  await session.hold(800);
  return modal;
};

/** createKey enters a key name, clicks Create key, and waits for the key to show. */
export const createKey = async (
  session: capture.CaptureSession,
  modal: Locator,
  name: string,
): Promise<void> => {
  await session.click(modal.getByPlaceholder("Key name", { exact: true }));
  await session.type(name);
  await session.hold(400);
  await session.click(modal.getByRole("button", { name: "Create key", exact: true }));
  await session.waitFor(modal.getByText("Copy this key now", { exact: false }));
  await session.waitFor(modal.getByText(name, { exact: true }));
  await session.hold(1800);
};
