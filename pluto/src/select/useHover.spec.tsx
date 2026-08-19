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
import { List } from "@/list";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

describe("useHover", () => {
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
        <List.Frame data={data}>
          <Triggers.Provider>{children}</Triggers.Provider>
        </List.Frame>
      );
      return dialog ? <Dialog.Frame visible>{content}</Dialog.Frame> : content;
    };
    return renderHook(
      () => Select.useHover({ data, onSelect, initialHover, enableTriggers }),
      { wrapper },
    );
  };

  const keyDown = (code: string): void => {
    fireEvent.keyDown(window, { code });
  };
  const keyUp = (code: string): void => {
    fireEvent.keyUp(window, { code });
  };

  it("should shift the hover position of the list when the down arrow is pressed", () => {
    const { result } = renderHover(DATA, vi.fn());
    keyDown("ArrowDown");
    expect(result.current.hover).toBe("alpha");
  });

  it("should accept an initial hover value", () => {
    const { result } = renderHover(DATA, vi.fn(), { initialHover: 1 });
    expect(result.current.hover).toBe("bravo");
  });

  it("should shift the hover position of the list when the up arrow is pressed", () => {
    const { result } = renderHover(DATA, vi.fn(), { initialHover: 1 });
    keyDown("ArrowUp");
    expect(result.current.hover).toBe("alpha");
  });

  it("should select the item when the enter key is pressed", () => {
    const onSelect = vi.fn();
    renderHover(DATA, onSelect, { initialHover: 1 });
    keyDown("Enter");
    expect(onSelect).toHaveBeenCalledWith("bravo");
  });

  it("should move the hover index to 0 when the initial hover is beyond the length of the list", () => {
    const { result } = renderHover(DATA, vi.fn(), { initialHover: 10 });
    expect(result.current.hover).toBe("alpha");
  });

  describe("enableTriggers", () => {
    it("should ignore keyboard triggers outside a dialog by default", () => {
      const onSelect = vi.fn();
      const { result } = renderHover(DATA, onSelect, {
        initialHover: 0,
        dialog: false,
      });
      keyDown("ArrowDown");
      expect(result.current.hover).toBe("alpha");
      keyDown("Enter");
      expect(onSelect).not.toHaveBeenCalled();
    });

    it("should answer keyboard triggers outside a dialog when enabled", () => {
      const onSelect = vi.fn();
      const { result } = renderHover(DATA, onSelect, {
        initialHover: 0,
        dialog: false,
        enableTriggers: true,
      });
      keyDown("ArrowDown");
      keyUp("ArrowDown");
      expect(result.current.hover).toBe("bravo");
      keyDown("Enter");
      expect(onSelect).toHaveBeenCalledWith("bravo");
    });

    it("should resolve a condition getter when the trigger fires", () => {
      let enabled = false;
      const { result } = renderHover(DATA, vi.fn(), {
        initialHover: 0,
        dialog: false,
        enableTriggers: () => enabled,
      });
      keyDown("ArrowDown");
      keyUp("ArrowDown");
      expect(result.current.hover).toBe("alpha");
      enabled = true;
      keyDown("ArrowDown");
      expect(result.current.hover).toBe("bravo");
    });
  });
});
