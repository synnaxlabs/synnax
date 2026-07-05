// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, configureStore } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { act, render, renderHook, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it, vi } from "vitest";

import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { Palette } from "@/palette";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const layoutFor = (key: string): Layout.PlacerArgs => ({
  key,
  type: "cat",
  name: `Layout ${key}`,
  location: "mosaic",
});

const createCommandsWrapper = (commands: Palette.Command[]) => {
  const store = configureStore({
    reducer: combineReducers({
      [Layout.SLICE_NAME]: Layout.reducer,
      [Drift.SLICE_NAME]: Drift.reducer,
    }),
  });
  const Base = createSynnaxWrapper({ client: null });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Base>
      <Provider store={store}>
        <Session.Modals.Provider>
          <Palette.CommandProvider commands={commands}>
            {children}
          </Palette.CommandProvider>
        </Session.Modals.Provider>
      </Provider>
    </Base>
  );
  return { store, wrapper };
};

interface CommandListHarness {
  handleSelect: (key: string) => void;
}

const renderCommandList = (commands: Palette.Command[]) => {
  const { store, wrapper } = createCommandsWrapper(commands);
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
    const useVisible = () => true;
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => () => {},
      sortOrder: 2,
      useVisible,
    });
    expect(cmd.key).toBe("cc");
    expect(cmd.commandName).toBe("Hook Command");
    expect(cmd.sortOrder).toBe(2);
    expect(cmd.useVisible).toBe(useVisible);
  });

  it("should invoke the hook-produced callback when the command is selected", () => {
    const onSelect = vi.fn();
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => onSelect,
    });
    const { harness } = renderCommandList([cmd]);
    expect(screen.getByText("Hook Command")).toBeTruthy();
    act(() => harness.handleSelect("cc"));
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("should not fire the callback for real pointer clicks handled by the list", () => {
    const onSelect = vi.fn();
    const cmd = Palette.createCommand({
      key: "cc",
      name: "Hook Command",
      useOnSelect: () => onSelect,
    });
    renderCommandList([cmd]);
    const item = screen.getByText("Hook Command");
    act(() => {
      item.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
    });
    expect(onSelect).not.toHaveBeenCalled();
  });
});

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

  it("should place its layout in the store when selected", () => {
    const cmd = Palette.createSimpleCommand({
      key: "sc",
      name: "Simple Command",
      layout: layoutFor("sc"),
    });
    const { store, harness } = renderCommandList([cmd]);
    act(() => harness.handleSelect("sc"));
    const placed = Layout.select(store.getState(), "sc");
    expect(placed?.type).toBe("cat");
    expect(placed?.location).toBe("mosaic");
  });
});

describe("useCommandContext", () => {
  it("should expose the commands supplied to the provider", () => {
    const commands = [
      Palette.createSimpleCommand({ key: "a", name: "Alpha", layout: layoutFor("a") }),
    ];
    const { wrapper } = createCommandsWrapper(commands);
    const { result } = renderHook(() => Palette.useCommandContext(), { wrapper });
    expect(result.current).toBe(commands);
  });
});

describe("useCommandList", () => {
  it("should filter out invisible commands and sort the rest by name", () => {
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
    const { wrapper } = createCommandsWrapper(commands);
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["c", "a"]);
  });

  it("should order commands by sortOrder when present", () => {
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
    const { wrapper } = createCommandsWrapper(commands);
    const { result } = renderHook(() => Palette.useCommandList(), { wrapper });
    expect(result.current.data).toEqual(["z", "y"]);
  });
});
