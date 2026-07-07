// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Import } from "@/feature/import";
import { interceptFilePicker, stubGeometry } from "@/testutil";

stubGeometry();

describe("Import Commands", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should open a JSON file picker when the import command is selected", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: Import.COMMANDS,
    });
    const picker = interceptFilePicker();
    await openCommandPalette();
    const item = await screen.findByText("Import component(s)");
    // The picker interceptor swallows programmatic clicks, including the select
    // frame's synthetic click, so fire the detail-0 click that invokes onSelect
    // directly.
    await act(async () => {
      fireEvent.click(item, { detail: 0 });
    });
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    const input = picker.lastInput();
    expect(input.accept).toBe(".json");
    expect(input.multiple).toBe(true);
    picker.cancel();
  });
});
