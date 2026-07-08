// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { renderPalette } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

const createScanTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_scan_rack") });
  return await rack.createTask(
    {
      name: uniqueName("ni_scanner"),
      type: NI.Task.SCAN_TYPE,
      config: { enabled: true },
    },
    NI.Task.SCAN_SCHEMAS,
  );
};

describe("palette", () => {
  it("should list every NI command once task-create access resolves", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: NI.Task.COMMANDS,
      client,
    });
    await openCommandPalette("NI");
    for (const name of [
      "Create an NI Analog Read Task",
      "Create an NI Analog Write Task",
      "Create an NI Counter Read Task",
      "Create an NI Digital Write Task",
      "Create an NI Digital Read Task",
      "Toggle the NI Device Scanner",
    ])
      await waitFor(() => expect(document.body.textContent).toContain(name));
  });

  it("should place the analog read layout when its command is selected", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: NI.Task.COMMANDS,
      client,
    });
    await openCommandPalette("Analog Read");
    await selectCommand("Create an NI Analog Read Task");
    await waitFor(() => {
      const placed = Session.Layout.selectByFilter(
        store.getState(),
        (l) => l.type === NI.Task.ANALOG_READ_TYPE,
      );
      if (placed == null) throw new Error("analog read layout not placed");
      expect(placed.name).toBe(NI.Task.ZERO_ANALOG_READ_PAYLOAD.name);
    });
  });

  it("should toggle the scanner task's enabled flag when the toggle command runs", async () => {
    await createScanTask();
    const before = await client.tasks.retrieve({
      types: [NI.Task.SCAN_TYPE],
      schemas: NI.Task.SCAN_SCHEMAS,
    });
    const target = before[0];
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: NI.Task.COMMANDS,
      client,
    });
    await openCommandPalette("Scanner");
    await selectCommand("Toggle the NI Device Scanner");
    await waitFor(async () => {
      const after = await client.tasks.retrieve({
        key: target.key,
        schemas: NI.Task.SCAN_SCHEMAS,
      });
      expect(after.config.enabled).toBe(!target.config.enabled);
    });
  });
});
