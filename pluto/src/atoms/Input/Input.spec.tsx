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
