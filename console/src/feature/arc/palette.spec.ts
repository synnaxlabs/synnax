// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { renderPalette } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { stubGeometry, uniqueName, waitForPlacedLayout } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("arc palette", () => {
  it("should place the explorer layout from the open explorer command", async () => {
    const { store, openCommandPalette } = await renderPalette({
      commands: Arc.COMMANDS,
      client,
    });
    await openCommandPalette();
    const item = await screen.findByText("Open the Arc Explorer");
    await act(async () => {
      fireEvent.click(item);
    });
    await waitForPlacedLayout(store, Arc.EXPLORER_LAYOUT_TYPE);
  });

  it("should create an arc through the create command's modal", async () => {
    const { store, openCommandPalette } = await renderPalette({
      commands: Arc.COMMANDS,
      client,
    });
    await openCommandPalette();
    const item = await screen.findByText("Create an Arc automation");
    await act(async () => {
      fireEvent.click(item);
    });
    const inputs = await screen.findAllByPlaceholderText("Automation Name");
    const input = inputs[inputs.length - 1];
    const name = uniqueName("arc");
    fireEvent.change(input, { target: { value: name } });
    const buttons = screen.getAllByRole("button", { name: "Create" });
    const create = buttons[buttons.length - 1];
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    await act(async () => {
      fireEvent.click(create);
    });
    await waitFor(() => {
      const placed = Session.Layout.selectByFilter(
        store.getState(),
        (l) => l.type === Arc.EDITOR_LAYOUT_TYPE && l.name === name,
      );
      if (placed == null) throw new Error(`no arc layout named ${name}`);
    });
  });
});
