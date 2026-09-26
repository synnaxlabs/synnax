// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

describe("Select keyboard hover", () => {
  const DATA = ["alpha", "bravo", "charlie"];

  interface RenderHoverOptions {
    initialHover?: number;
    enableTriggers?: Triggers.Condition;
    dialog?: boolean;
  }

  const renderHover = (
    data: string[],
    onSelect: (key: string) => void,
    { initialHover, enableTriggers, dialog = true }: RenderHoverOptions = {},
  ) => {
    const wrapper = ({ children }: PropsWithChildren): ReactElement => {
      const content = (
        <Triggers.Provider>
          <Select.Frame
            data={data}
            onChange={(key: string) => onSelect(key)}
            initialHover={initialHover}
            enableTriggers={enableTriggers}
          >
            {children}
          </Select.Frame>
        </Triggers.Provider>
      );
      return dialog ? <Dialog.Frame visible>{content}</Dialog.Frame> : content;
    };
    const { result } = renderHook(() => Select.useContext<string>(), { wrapper });
    return { hover: () => result.current.getState().hover };
  };

  const keyDown = (code: string): void => {
    fireEvent.keyDown(window, { code });
  };
  const keyUp = (code: string): void => {
    fireEvent.keyUp(window, { code });
  };

  it("should shift the hover position of the list when the down arrow is pressed", () => {
    const { hover } = renderHover(DATA, vi.fn());
    keyDown("ArrowDown");
    expect(hover()).toBe("alpha");
  });

  it("should accept an initial hover value", () => {
    const { hover } = renderHover(DATA, vi.fn(), { initialHover: 1 });
    expect(hover()).toBe("bravo");
  });

  it("should shift the hover position of the list when the up arrow is pressed", () => {
    const { hover } = renderHover(DATA, vi.fn(), { initialHover: 1 });
    keyDown("ArrowUp");
    expect(hover()).toBe("alpha");
  });

  it("should select the item when the enter key is pressed", () => {
    const onSelect = vi.fn();
    renderHover(DATA, onSelect, { initialHover: 1 });
    keyDown("Enter");
    expect(onSelect).toHaveBeenCalledWith("bravo");
  });

  it("should move the hover index to 0 when the initial hover is beyond the length of the list", () => {
    const { hover } = renderHover(DATA, vi.fn(), { initialHover: 10 });
    expect(hover()).toBe("alpha");
  });

  describe("enableTriggers", () => {
    it("should ignore keyboard triggers outside a dialog by default", () => {
      const onSelect = vi.fn();
      const { hover } = renderHover(DATA, onSelect, {
        initialHover: 0,
        dialog: false,
      });
      keyDown("ArrowDown");
      expect(hover()).toBe("alpha");
      keyDown("Enter");
      expect(onSelect).not.toHaveBeenCalled();
    });

    it("should answer keyboard triggers outside a dialog when enabled", () => {
      const onSelect = vi.fn();
      const { hover } = renderHover(DATA, onSelect, {
        initialHover: 0,
        dialog: false,
        enableTriggers: true,
      });
      keyDown("ArrowDown");
      keyUp("ArrowDown");
      expect(hover()).toBe("bravo");
      keyDown("Enter");
      expect(onSelect).toHaveBeenCalledWith("bravo");
    });

    it("should resolve a condition getter when the trigger fires", () => {
      let enabled = false;
      const { hover } = renderHover(DATA, vi.fn(), {
        initialHover: 0,
        dialog: false,
        enableTriggers: () => enabled,
      });
      keyDown("ArrowDown");
      keyUp("ArrowDown");
      expect(hover()).toBe("alpha");
      enabled = true;
      keyDown("ArrowDown");
      expect(hover()).toBe("bravo");
    });
  });
});
