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
import { describe, expect, it, vi } from "vitest";

import { Palette } from "@/app/palette";
import { Command as FeatureCommand } from "@/feature/command";
import { Docs } from "@/feature/docs";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, selectTestProject } from "@/testutil";

const renderAppPalette = async () => {
  const client = createTestClient();
  const { wrapper, store } = await createConsoleWrapper({ client });
  await selectTestProject(store, client);
  render(
    <>
      <Palette.Palette />
      <Modals.Stack />
    </>,
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

/** Presses the codes as a chord, holding every key down before releasing them. */
const pressChord = async (...codes: string[]): Promise<HTMLInputElement> => {
  await act(async () => {
    codes.forEach((code) => fireEvent.keyDown(window, { code }));
  });
  await act(async () => {
    codes.forEach((code) => fireEvent.keyUp(window, { code }));
  });
  return await waitFor(() => {
    const input = document.querySelector<HTMLInputElement>(
      ".console-palette__input input",
    );
    if (input == null) throw new Error("palette input not found");
    return input;
  });
};

describe("Palette", () => {
  it("should open in command mode from its keyboard trigger", async () => {
    await renderAppPalette();
    const input = await pressChord("ControlLeft", "ShiftLeft", "KeyP");
    expect(input.value).toBe(FeatureCommand.PREFIX);
  });

  it("should open in search mode from its keyboard trigger", async () => {
    await renderAppPalette();
    const input = await pressChord("ControlLeft", "KeyP");
    expect(input.value).toBe("");
  });

  it("should run a command selected through the command palette", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    await renderAppPalette();
    const input = await openPalette();
    fireEvent.change(input, { target: { value: ">Read the documentation" } });
    const item = await screen.findByText("Read documentation");
    await act(async () => {
      fireEvent.click(item, { detail: 1 });
    });
    await waitFor(() => {
      expect(open).toHaveBeenCalledWith(Docs.URL, "_blank", "noopener,noreferrer");
    });
  });
});
