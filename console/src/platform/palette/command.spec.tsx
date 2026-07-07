// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { act, render, renderHook, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Palette } from "@/platform/palette";
import { createPaletteWrapper } from "@/platform/palette/testutil";
import { renderHookWithConsole } from "@/testutil";

interface CommandListHarness {
  handleSelect: (key: string) => void;
}

const renderCommandList = async (
  commands: Palette.Command[],
  client: Synnax | null = null,
) => {
  const { wrapper, store } = await createPaletteWrapper({ commands, client });
  const harness: CommandListHarness = { handleSelect: () => {} };
  const Items = (): ReactElement => {
    const { data, listItem, handleSelect } = Palette.useCommandList();
    harness.handleSelect = handleSelect;
    return <>{data.map((key, index) => listItem({ key, itemKey: key, index }))}</>;
  };
  render(<Items />, { wrapper });
  return { store, harness };
};

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

  it("should forward sortOrder and useVisible onto the command", () => {
    const useVisible = () => true;
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => () => {},
      sortOrder: 3,
      useVisible,
    });
    expect(cmd.sortOrder).toBe(3);
    expect(cmd.useVisible).toBe(useVisible);
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
    // Must be invoked arg-less so the click event never leaks into a callback that
    // takes optional args (e.g. a resource create hook).
    expect(onSelect).toHaveBeenCalledWith();
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
      Palette.createCommand({ key: "a", name: "Alpha", useOnSelect: () => () => {} }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandContext(), { wrapper });
    expect(result.current).toBe(commands);
  });
});

describe("useCommandList", () => {
  it("should filter out invisible commands and sort the rest", async () => {
    const commands = [
      Palette.createCommand({ key: "a", name: "Alpha", useOnSelect: () => () => {} }),
      Palette.createCommand({
        key: "b",
        name: "Beta",
        useOnSelect: () => () => {},
        useVisible: () => false,
      }),
      Palette.createCommand({
        key: "c",
        name: "Aardvark",
        useOnSelect: () => () => {},
      }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["c", "a"]);
  });

  it("should order commands by sortOrder when present", async () => {
    const commands = [
      Palette.createCommand({
        key: "z",
        name: "Zeta",
        useOnSelect: () => () => {},
        sortOrder: 1,
      }),
      Palette.createCommand({
        key: "y",
        name: "Yotta",
        useOnSelect: () => () => {},
        sortOrder: 2,
      }),
    ];
    const { wrapper } = await createPaletteWrapper({ commands });
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["z", "y"]);
  });
});
