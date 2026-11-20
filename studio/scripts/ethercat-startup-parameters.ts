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
  await fill("Subindex", "15");
  await session.hold(300);
  // The trigger's accessible name also holds its value, so match its own label.
  await session.click(row.locator("[aria-label='Data type 1']").first());
  const option = session.page
    .locator(".lyra-select__dialog:visible .lyra-list__item")
    .filter({ hasText: /^\s*uint16\s*$/ })
    .first();
  await session.waitFor(option);
  await session.hold(500);
  await session.click(option, { text: true });
  await session.hold(400);
  await fill("Value", "1");
  // Tab from the last cell focuses the remove button and shows its tooltip.
  await session.click(cell("Name"));
  await session.type("Filter settings");
  await session.hold(1800);
});
