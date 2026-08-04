// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Palette } from "@/app/palette";
import { Docs } from "@/feature/docs";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, resolveFocusedTab, selectTestProject } from "@/testutil";

const renderAppPalette = async () => {
  const client = createTestClient();
  const { wrapper, store } = await createConsoleWrapper({ client });
  await selectTestProject(store, client);
  render(
    <Import.FileIngestersProvider fileIngesters={{}}>
      <Palette.Palette />
      <Modals.Stack />
    </Import.FileIngestersProvider>,
    { wrapper },
  );
  return { store, client };
};

const openPalette = async (): Promise<HTMLInputElement> => {
  const btn = document.querySelector<HTMLElement>(".console-palette__btn");
  if (btn == null) throw new Error("palette open button not found");
  fireEvent.click(btn);
  return await waitFor(() => {
    const input = document.querySelector<HTMLInputElement>(
      ".console-palette__input input",
    );
    if (input == null) throw new Error("palette input not found");
    return input;
  });
};

describe("Palette", () => {
  it("should run a command selected through the command palette", async () => {
    const { store, client } = await renderAppPalette();
    const input = await openPalette();
    fireEvent.change(input, { target: { value: ">Read the documentation" } });
    const item = await screen.findByText("Read the documentation");
    await act(async () => {
      fireEvent.click(item, { detail: 1 });
    });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Docs.TAB_TYPE);
  });
});
