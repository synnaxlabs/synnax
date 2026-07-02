// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { type Layout } from "@/platform/layout";
import { Palette } from "@/platform/palette";
import { createPaletteWrapper, TIMEOUT } from "@/platform/palette/testutil";
import { Session } from "@/session";

const TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

const layoutFor = (key: string): Layout.PlacerArgs => ({
  key,
  type: "cat",
  name: `Layout ${key}`,
  location: "mosaic",
});

const renderPalette = async (commands: Palette.Command[]) => {
  const { wrapper, store } = await createPaletteWrapper({ commands });
  const result = render(
    <Palette.Palette commandSymbol=">" triggerConfig={TRIGGER_CONFIG} />,
    { wrapper },
  );
  return { ...result, store };
};

const openButton = (): HTMLElement => {
  const btn = document.querySelector<HTMLElement>(".console-palette__btn");
  if (btn == null) throw new Error("palette open button not found");
  return btn;
};

const paletteInput = (): HTMLInputElement => {
  const input = document.querySelector<HTMLInputElement>("input");
  if (input == null) throw new Error("palette input not found");
  return input;
};

describe("Palette", () => {
  it("should render a closed palette with an open button", async () => {
    await renderPalette([]);
    expect(openButton()).toBeTruthy();
    expect(document.querySelector(".console-palette__input")).toBeNull();
  });

  it("should open the dialog and reveal the search input", async () => {
    await renderPalette([]);
    fireEvent.click(openButton());
    await waitFor(() => expect(paletteInput()).toBeTruthy(), TIMEOUT);
  });

  it("should list the provided commands in command mode", async () => {
    const commands = [
      Palette.createSimpleCommand({
        key: "a",
        name: "Create Alpha",
        layout: layoutFor("a"),
      }),
      Palette.createSimpleCommand({
        key: "b",
        name: "Create Beta",
        layout: layoutFor("b"),
      }),
      Palette.createCommand({
        key: "c",
        name: "Hook Command",
        useOnSelect: () => () => {},
      }),
    ];
    await renderPalette(commands);
    fireEvent.click(openButton());
    await waitFor(() => expect(paletteInput()).toBeTruthy(), TIMEOUT);
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    await waitFor(() => {
      expect(screen.getByText("Create Alpha")).toBeTruthy();
      expect(screen.getByText("Create Beta")).toBeTruthy();
      expect(screen.getByText("Hook Command")).toBeTruthy();
    }, TIMEOUT);
  });

  it("should place a command's layout and close the dialog when selected", async () => {
    const commands = [
      Palette.createSimpleCommand({
        key: "a",
        name: "Create Alpha",
        layout: layoutFor("a"),
      }),
    ];
    const { store } = await renderPalette(commands);
    fireEvent.click(openButton());
    await waitFor(() => expect(paletteInput()).toBeTruthy(), TIMEOUT);
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    const item = await screen.findByText("Create Alpha");
    await act(async () => {
      fireEvent.click(item);
    });
    await waitFor(
      () => expect(Session.Layout.select(store.getState(), "a")?.name).toBe("Layout a"),
      TIMEOUT,
    );
  });

  it("should show the empty command state when no commands are visible", async () => {
    await renderPalette([]);
    fireEvent.click(openButton());
    await waitFor(() => expect(paletteInput()).toBeTruthy(), TIMEOUT);
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    await waitFor(
      () => expect(screen.getByText("No commands found")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should show the empty resource state when searching without matches", async () => {
    await renderPalette([]);
    fireEvent.click(openButton());
    await waitFor(() => expect(paletteInput()).toBeTruthy(), TIMEOUT);
    fireEvent.change(paletteInput(), { target: { value: "no-such-resource" } });
    await waitFor(
      () => expect(screen.getByText("No resources found")).toBeTruthy(),
      TIMEOUT,
    );
  });
});
