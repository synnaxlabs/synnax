// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Label } from "@/feature/label";
import { stubGeometry } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("Label Commands", () => {
  it("should open the label edit modal when the command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Label.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Edit labels");
    expect(await screen.findByText("Search labels")).toBeTruthy();
  });
});
