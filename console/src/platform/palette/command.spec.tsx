// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render, renderHook, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { type Layout } from "@/platform/layout";
import { Palette } from "@/platform/palette";
import { createPaletteWrapper } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

const layoutFor = (key: string): Layout.PlacerArgs => ({
  key,
  type: "cat",
  name: `Layout ${key}`,
  location: "mosaic",
});

interface CommandListHarness {
  handleSelect: (key: string) => void;
}

const renderCommandList = async (commands: Palette.Command[]) => {
  const { wrapper, store } = await createPaletteWrapper({ commands });
  const harness: CommandListHarness = { handleSelect: () => {} };
  const Items = (): ReactElement => {
    const { data, listItem, handleSelect } = Palette.useCommandList();
    harness.handleSelect = handleSelect;
    return <>{data.map((key, index) => listItem({ key, itemKey: key, index }))}</>;
  };
  render(<Items />, { wrapper });
  return { store, harness };
};

describe("createSimpleCommand", () => {
  it("should attach the static command metadata", () => {
    const useVisible = () => true;
    const cmd = Palette.createSimpleCommand({
      key: "sc",
      name: "Simple Command",
      layout: layoutFor("sc"),
      sortOrder: 3,
      useVisible,
    });
    expect(cmd.key).toBe("sc");
    expect(cmd.commandName).toBe("Simple Command");
    expect(cmd.sortOrder).toBe(3);
    expect(cmd.useVisible).toBe(useVisible);
  });

  it("should place its layout in the store when selected", async () => {
    const cmd = Palette.createSimpleCommand({
      key: "sc",
      name: "Simple Command",
      layout: layoutFor("sc"),
    });
    const { store, harness } = await renderCommandList([cmd]);
    act(() => harness.handleSelect("sc"));
    const placed = Session.Layout.select(store.getState(), "sc");
    expect(placed?.type).toBe("cat");
    expect(placed?.location).toBe("mosaic");
  });
});

describe("createCommand", () => {
  it("should attach the static command metadata", () => {
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => () => {},
    });
    expect(cmd.key).toBe("cc");
    expect(cmd.commandName).toBe("Hook Command");
    expect(cmd.sortOrder).toBeUndefined();
    expect(cmd.useVisible).toBeUndefined();
  });

  it("should invoke the hook-produced callback when the command is selected", async () => {
    const onSelect = vi.fn();
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => onSelect,
    });
    const { harness } = await renderCommandList([cmd]);
    expect(screen.getByText("Hook Command")).toBeTruthy();
    act(() => harness.handleSelect("cc"));
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("should not fire the callback for real pointer clicks handled by the list", async () => {
    const onSelect = vi.fn();
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => onSelect,
    });
    await renderCommandList([cmd]);
    const item = screen.getByText("Hook Command");
    act(() => {
      item.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
    });
    expect(onSelect).not.toHaveBeenCalled();
  });
});

describe("useCommandContext", () => {
  it("should default to an empty command list without a provider", async () => {
    const { result } = await renderHookWithConsole(() => Palette.useCommandContext());
    expect(result.current).toHaveLength(0);
  });

  it("should expose the commands supplied to the provider", async () => {
    const commands = [
      Palette.createSimpleCommand({ key: "a", name: "Alpha", layout: layoutFor("a") }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandContext(), { wrapper });
    expect(result.current).toBe(commands);
  });
});

describe("useCommandList", () => {
  it("should filter out invisible commands and sort the rest", async () => {
    const commands = [
      Palette.createSimpleCommand({ key: "a", name: "Alpha", layout: layoutFor("a") }),
      Palette.createSimpleCommand({
        key: "b",
        name: "Beta",
        layout: layoutFor("b"),
        useVisible: () => false,
      }),
      Palette.createSimpleCommand({
        key: "c",
        name: "Aardvark",
        layout: layoutFor("c"),
      }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["c", "a"]);
  });

  it("should order commands by sortOrder when present", async () => {
    const commands = [
      Palette.createSimpleCommand({
        key: "z",
        name: "Zeta",
        layout: layoutFor("z"),
        sortOrder: 1,
      }),
      Palette.createSimpleCommand({
        key: "y",
        name: "Yotta",
        layout: layoutFor("y"),
        sortOrder: 2,
      }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["z", "y"]);
  });
});
