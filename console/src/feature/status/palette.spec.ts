// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/feature/status";
import { findModalButton } from "@/platform/ontology/menuTestutil";
import { renderPalette } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { stubGeometry } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("status palette", () => {
  it("should open the status creation modal when the create command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Status.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Create a status");
    await screen.findByRole("dialog");
    expect(findModalButton("Create")).toBeTruthy();
  });

  it("should place the status explorer layout when the explorer command is selected", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Status.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Open the Status Explorer");
    await waitFor(() =>
      expect(
        Session.Layout.select(store.getState(), Status.EXPLORER_LAYOUT_TYPE)?.type,
      ).toBe(Status.EXPLORER_LAYOUT_TYPE),
    );
  });
});
