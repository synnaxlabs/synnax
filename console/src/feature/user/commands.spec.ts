// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { User } from "@/feature/user";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { stubGeometry } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("User Commands", () => {
  it("should open the register modal when the command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: User.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Register a user");
    await screen.findByRole("dialog");
    expect(findModalButton("Register")).toBeTruthy();
  });
});
