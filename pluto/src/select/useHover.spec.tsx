// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

describe("useHover", () => {
  const renderHover = (
    data: string[],
    onSelect: (key: string) => void,
    initialHover?: number,
  ) => {
    const C = () => {
      const { hover } = Select.useHover({
        data,
        onSelect,
        initialHover,
      });
      return <div>{hover}</div>;
    };
    return render(
      <Dialog.Frame visible>
        <List.Frame data={data}>
          <Triggers.Provider>
            <C />
          </Triggers.Provider>
        </List.Frame>
      </Dialog.Frame>,
    );
  };

  it("should shift the hover position of the list when the down arrow is pressed", () => {
    const onSelect = vi.fn();
    const data = ["1", "2", "3"];
    const c = renderHover(data, onSelect);

    fireEvent.keyDown(c.container, { code: "ArrowDown" });
    expect(c.getByText("1")).toBeTruthy();
  });

  it("should accept an initial hover value", () => {
    const onSelect = vi.fn();
    const data = ["1", "2", "3"];
    const c = renderHover(data, onSelect, 1);

    expect(c.getByText("2")).toBeTruthy();
  });

  it("should shift the hover position of the list when the up arrow is pressed", () => {
    const onSelect = vi.fn();
    const data = ["1", "2", "3"];
    const c = renderHover(data, onSelect, 1);
    fireEvent.keyDown(c.container, { code: "ArrowUp" });
    expect(c.getByText("1")).toBeTruthy();
  });

  it("should select the item when the enter key is pressed", () => {
    const onSelect = vi.fn();
    const data = ["1", "2", "3"];
    const c = renderHover(data, onSelect, 1);
    fireEvent.keyDown(c.container, { code: "Enter" });
    expect(onSelect).toHaveBeenCalledWith("2");
  });

  it("should move the hover index to 0 when the initial hover is beyond the length of the list", () => {
    const onSelect = vi.fn();
    const data = ["1", "2", "3"];
    const c = renderHover(data, onSelect, 10);
    expect(c.getByText("1")).toBeTruthy();
  });
});
