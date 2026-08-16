// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Selector } from "@/platform/selector";

describe("Selector.createSelectable", () => {
  it("invokes the bound callback with no arguments when clicked", () => {
    const onSelect = vi.fn();
    const C = Selector.createSelectable({
      type: "cat",
      title: "Cat",
      icon: <Icon.Add />,
      useOnSelect: () => onSelect,
    });
    render(<C />);
    fireEvent.click(screen.getByText("Cat"));
    expect(onSelect).toHaveBeenCalledTimes(1);
    // Must be invoked arg-less so the click event never leaks into a callback that
    // takes optional args (e.g. a resource create hook).
    expect(onSelect).toHaveBeenCalledWith();
  });

  it("renders nothing when useVisible returns false", () => {
    const C = Selector.createSelectable({
      type: "cat",
      title: "Hidden",
      icon: <Icon.Add />,
      useOnSelect: () => vi.fn(),
      useVisible: () => false,
    });
    render(<C />);
    expect(screen.queryByText("Hidden")).toBeNull();
  });
});
