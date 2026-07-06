// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ontology } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { type Layout } from "@/platform/layout";
import { Ontology } from "@/platform/ontology";
import { createServices } from "@/platform/ontology/testutil";
import { Palette } from "@/platform/palette";
import {
  createPaletteWrapper,
  type CreatePaletteWrapperArgs,
} from "@/platform/palette/testutil";
import { Session } from "@/session";
import { stubGeometry } from "@/testutil";

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

const renderPalette = async (args: CreatePaletteWrapperArgs = {}) => {
  const { wrapper, store } = await createPaletteWrapper(args);
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

const openPalette = async (): Promise<void> => {
  fireEvent.click(openButton());
  await waitFor(() => expect(paletteInput()).toBeTruthy());
};

stubGeometry();

describe("Palette", () => {
  it("should filter the command list down to fuzzy matches of the typed text", async () => {
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
    ];
    await renderPalette({ commands });
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    await waitFor(() => {
      expect(screen.getByText("Create Alpha")).toBeTruthy();
      expect(screen.getByText("Create Beta")).toBeTruthy();
    });
    fireEvent.change(paletteInput(), { target: { value: ">Alpha" } });
    await waitFor(() => {
      expect(screen.getByText("Create Alpha")).toBeTruthy();
      expect(screen.queryByText("Create Beta")).toBeNull();
    });
  });

  it("should place a command's layout and close the dialog when selected", async () => {
    const commands = [
      Palette.createSimpleCommand({
        key: "a",
        name: "Create Alpha",
        layout: layoutFor("a"),
      }),
    ];
    const { store } = await renderPalette({ commands });
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    const item = await screen.findByText("Create Alpha");
    await act(async () => {
      fireEvent.click(item);
    });
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), "a")?.name).toBe("Layout a"),
    );
    await waitFor(() =>
      expect(document.querySelector(".console-palette__input")).toBeNull(),
    );
  });

  it("should invoke a hook command's onSelect callback when selected", async () => {
    const onSelect = vi.fn();
    const commands = [
      Palette.createCommand({
        key: "hook",
        name: "Run Hook",
        useOnSelect: () => onSelect,
      }),
    ];
    await renderPalette({ commands });
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    const item = await screen.findByText("Run Hook");
    await act(async () => {
      fireEvent.click(item, { detail: 1 });
    });
    await waitFor(() => expect(onSelect).toHaveBeenCalledTimes(1));
  });

  it("should show the empty command state when no commands are visible", async () => {
    await renderPalette();
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: ">" } });
    await waitFor(() => expect(screen.getByText("No commands found")).toBeTruthy());
  });

  it("should show the empty resource state when searching without matches", async () => {
    await renderPalette();
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: "no_such_resource" } });
    await waitFor(() => expect(screen.getByText("No resources found")).toBeTruthy());
  });

  it("should search the live cluster for resources and invoke the service's onSelect", async () => {
    const client = createTestClient();
    const name = `pal_res_${id.create().replace(/-/g, "_")}`;
    await client.channels.create({ name, dataType: "float32", virtual: true });
    await waitFor(async () => {
      const res = await client.ontology.retrieve({ searchTerm: name });
      expect(res.some((r) => r.name === name)).toBe(true);
    });

    const onSelect = vi.fn();
    const services = createServices({
      channel: { ...Ontology.NOOP_SERVICE, type: "channel", onSelect },
    });
    await renderPalette({ client, services });
    await openPalette();
    fireEvent.change(paletteInput(), { target: { value: name } });
    const item = await screen.findByText(name);
    await act(async () => {
      fireEvent.click(item);
    });
    await waitFor(() => expect(onSelect).toHaveBeenCalledTimes(1));
    const selection = onSelect.mock.calls[0][0].selection as ontology.Resource[];
    expect(selection.map((r) => r.name)).toContain(name);
  });
});
