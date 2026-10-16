// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type KeyboardEvent } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Select } from "@/select";
import { Triggers } from "@/triggers";

const renderSearch = (
  onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void,
  onChange: (key: string | null) => void,
) =>
  render(
    <Triggers.Provider>
      <Dialog.Frame visible>
        <Select.Frame<string, undefined> onChange={onChange} allowNone initialHover={0}>
          <Select.Dialog>
            <Select.Search onKeyDown={onKeyDown} />
            <Select.List>
              <Select.Item itemKey="alpha">alpha</Select.Item>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Triggers.Provider>,
  );

describe("Select.Search", () => {
  it("should pass the field's keys to onKeyDown", () => {
    const onKeyDown = vi.fn();
    const c = renderSearch(onKeyDown, vi.fn());
    fireEvent.keyDown(c.getByPlaceholderText("Search..."), { key: "a", code: "KeyA" });
    expect(onKeyDown).toHaveBeenCalledWith(expect.objectContaining({ key: "a" }));
  });

  it("should let onKeyDown keep Enter from picking the hovered item", () => {
    const onChange = vi.fn();
    const c = renderSearch((e) => e.stopPropagation(), onChange);
    fireEvent.keyDown(c.getByPlaceholderText("Search..."), {
      key: "Enter",
      code: "Enter",
    });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should leave Enter to pick the hovered item by default", () => {
    const onChange = vi.fn();
    const c = renderSearch(vi.fn(), onChange);
    fireEvent.keyDown(c.getByPlaceholderText("Search..."), {
      key: "Enter",
      code: "Enter",
    });
    expect(onChange).toHaveBeenCalledWith("alpha", expect.anything());
  });
});
