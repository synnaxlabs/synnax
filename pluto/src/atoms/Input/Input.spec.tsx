// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Input } from ".";

describe("Input", () => {
  it("should render an input with the provided placeholder", () => {
    const c = render(<Input placeholder="Hello" onChange={vi.fn()} />);
    expect(c.getByPlaceholderText("Hello")).toBeTruthy();
  });
  it("should call the onChange handler when the value changes", () => {
    const onChange = vi.fn();
    const c = render(<Input placeholder="Hello" onChange={onChange} />);
    expect(onChange).not.toHaveBeenCalled();
    const input = c.getByPlaceholderText("Hello");
    fireEvent.change(input, { target: { value: "Hello" } });
    expect(onChange).toHaveBeenCalled();
  });
  it("should programatically set the input value when the prop is passed", () => {
    const c = render(<Input placeholder="Hello" onChange={vi.fn()} value="Hello2" />);
    expect(c.getByDisplayValue("Hello2")).toBeTruthy();
  });
});
