// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureScript } from "./common/ethercat";
import { fieldItem } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/configure-device` (Sync mode): in the Configure
 * dialog of a slave, open Mode, select the distributed clock mode, and set the SYNC0
 * cycle. The slave is simulated.
 */
export default configureScript(async (session, modal) => {
  const { page } = session;
  const mode = fieldItem(modal, "Mode");
  await mode.scrollIntoViewIfNeeded();
  await session.click(mode.locator(".lyra-dialog__trigger").first());
  const option = page
    .locator(".lyra-list__item:not(.lyra-tree__item)")
    .filter({ hasText: "DC-Synchron" })
    .first();
  await session.waitFor(option);
  await session.hold(2000);
  await session.click(option, { text: true });
  const cycle = modal.getByLabel("SYNC0 cycle", { exact: true });
  await session.waitFor(cycle);
  await session.hold(1000);
  await session.click(cycle);
  await session.press("ControlOrMeta+a");
  await session.type("1000");
  await session.press("Enter");
  await session.hold(2000);
});
