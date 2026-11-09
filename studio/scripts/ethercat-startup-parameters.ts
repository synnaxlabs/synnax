// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureScript } from "./common/ethercat";

/**
 * Docs `reference/hardware/ethercat/configure-device` (Startup parameters): in the
 * Configure dialog of a slave, add a CoE setting that the Driver writes each time the
 * network starts. The slave is simulated.
 */
export default configureScript(async (session, modal) => {
  const add = modal.getByRole("button", { name: "Add parameter", exact: true });
  await add.scrollIntoViewIfNeeded();
  await session.click(add);
  await session.hold(800);
  const row = modal.locator("tbody tr").first();
  const cell = (name: string) => row.getByLabel(`${name} 1`, { exact: true });
  await session.waitFor(cell("Index"));
  const fill = async (name: string, value: string): Promise<void> => {
    await session.click(cell(name));
    await session.press("ControlOrMeta+a");
    await session.type(value);
    await session.press("Tab");
    await session.hold(300);
  };
  await fill("Index", "8000");
  await fill("Subindex", "19");
  await session.hold(300);
  await fill("Value", "2");
  await fill("Name", "Filter settings");
  await session.hold(1800);
});
