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
import { Range } from "@/feature/range";

const client = createTestClient();

describe("Range Commands", () => {
  it("shows both commands to a user with range permissions", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: Range.COMMANDS,
      client,
    });
    await openCommandPalette("range");
    expect(await screen.findByText("Create range")).toBeTruthy();
    expect(await screen.findByText("Open range explorer")).toBeTruthy();
  });
});
