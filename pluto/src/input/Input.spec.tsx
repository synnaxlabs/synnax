// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";

describe("Input", () => {
  describe("Text", () => {
    it("should render an input with the provided placeholder", () => {
      const c = render(<Input.Text value="" placeholder="Hello" onChange={vi.fn()} />);
      expect(c.getByText("Hello")).toBeTruthy();
    });
    it("should call the onChange handler when the value changes", () => {
      const onChange = vi.fn();
      const c = render(<Input.Text value="" placeholder="Hello" onChange={onChange} />);
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByText("Hello").parentElement?.parentElement?.children[1];
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "Hello" } });
      expect(onChange).toHaveBeenCalled();
    });
    it("should programatically set the input value when the prop is passed", () => {
      const c = render(
        <Input.Text placeholder="Hello" onChange={vi.fn()} value="Hello2" />,
      );
      expect(c.getByDisplayValue("Hello2")).toBeTruthy();
    });
  });
  describe("Numeric", () => {
    it("should render an input with the provided placeholder", () => {
      const c = render(<Input.Numeric value={0} onChange={vi.fn()} />);
      expect(c.getByDisplayValue("0")).toBeTruthy();
    });
    it("should call the onchange handler when a valid number is input", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "1" } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).toHaveBeenCalledWith(1);
    });
    it("should not call the onchange handler when an invalid number is input", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "Hello" } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).not.toHaveBeenCalled();
    });
    it("should not call the onchange handler when the user inputs an empty value", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "" } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).not.toHaveBeenCalled();
    });
    it("should not call the onChange handler when the user inputs NaN", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "NaN" } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).not.toHaveBeenCalled();
    });
    it("should not call the onChange handler when the user inputs a *", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: '""' } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).not.toHaveBeenCalled();
    });
    it("should correctly evaluate a mathematical expression", () => {
      const onChange = vi.fn();
      const c = render(
        <Input.Numeric value={0} placeholder="Hello" onChange={onChange} />,
      );
      expect(onChange).not.toHaveBeenCalled();
      const input = c.getByDisplayValue("0");
      expect(input).not.toBeUndefined();
      fireEvent.change(input as HTMLInputElement, { target: { value: "1+1" } });
      fireEvent.blur(input as HTMLInputElement);
      expect(onChange).toHaveBeenCalledWith(2);
    });
  });
});
