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

import { Input } from "@/input";

describe("Input", () => {
  describe("Text", () => {
    describe("Basic Rendering", () => {
      it("should render a text input with the provided value", () => {
        const c = render(<Input.Text value="Hello" onChange={vi.fn()} />);
        const input = c.getByRole("textbox");
        expect(input).toBeTruthy();
        expect(input.tagName).toBe("INPUT");
        expect((input as HTMLInputElement).value).toBe("Hello");
      });

      it("should render with medium size by default", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-medium");
      });
    });

    describe("size", () => {
      it("should render a small input if the size is small", () => {
        const c = render(<Input.Text size="small" value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-small");
      });

      it("should render a large input if the size is large", () => {
        const c = render(<Input.Text size="large" value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-large");
      });

      it("should render a huge input if the size is huge", () => {
        const c = render(<Input.Text size="huge" value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-huge");
      });

      it("should render a tiny input if the size is tiny", () => {
        const c = render(<Input.Text size="tiny" value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-tiny");
      });
    });

    describe("variant", () => {
      it("should render an outlined input by default", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto-btn--outlined");
      });

      it("should render a text variant input", () => {
        const c = render(<Input.Text variant="text" value="" onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto-btn--text");
      });
    });

    describe("disabled", () => {
      it("should add the disabled class when disabled is true", () => {
        const c = render(<Input.Text disabled value="" onChange={vi.fn()} />);
        const input = c.getByRole("textbox");
        const container = input.parentElement;
        expect(container?.className).toContain("pluto--disabled");
        expect((input as HTMLInputElement).disabled).toBe(true);
      });

      it("should not call onChange when disabled", () => {
        const onChange = vi.fn();
        const c = render(<Input.Text disabled value="" onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "test" } });
        expect(onChange).not.toHaveBeenCalled();
      });
    });

    describe("onChange behavior", () => {
      it("should call onChange when input value changes", () => {
        const onChange = vi.fn();
        const c = render(<Input.Text value="" onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "test" } });
        expect(onChange).toHaveBeenCalledWith("test");
      });

      it("should not call onChange immediately when onlyChangeOnBlur is true", () => {
        const onChange = vi.fn();
        const c = render(<Input.Text value="" onChange={onChange} onlyChangeOnBlur />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "test" } });
        expect(onChange).not.toHaveBeenCalled();
      });

      it("should call onChange on blur when onlyChangeOnBlur is true", () => {
        const onChange = vi.fn();
        const c = render(<Input.Text value="" onChange={onChange} onlyChangeOnBlur />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "test" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith("test");
      });
    });

    describe("focus and blur handlers", () => {
      it("should call onFocus when input is focused", () => {
        const onFocus = vi.fn();
        const c = render(<Input.Text value="" onChange={vi.fn()} onFocus={onFocus} />);
        const input = c.getByRole("textbox");
        fireEvent.focus(input);
        expect(onFocus).toHaveBeenCalled();
      });

      it("should call onBlur when input loses focus", () => {
        const onBlur = vi.fn();
        const c = render(<Input.Text value="" onChange={vi.fn()} onBlur={onBlur} />);
        const input = c.getByRole("textbox");
        fireEvent.blur(input);
        expect(onBlur).toHaveBeenCalled();
      });
    });

    describe("resetOnBlurIfEmpty", () => {
      it("should reset to original value on blur if empty", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Text
            value="original"
            onChange={onChange}
            resetOnBlurIfEmpty
            onlyChangeOnBlur
          />,
        );
        const input = c.getByRole("textbox");
        fireEvent.focus(input);
        fireEvent.change(input, { target: { value: "" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith("original");
      });

      it("should not reset if input is not empty on blur", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Text value="original" onChange={onChange} resetOnBlurIfEmpty />,
        );
        const input = c.getByRole("textbox");
        fireEvent.focus(input);
        fireEvent.change(input, { target: { value: "new" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith("new");
      });
    });

    describe("placeholder", () => {
      it("should render a string placeholder", () => {
        const c = render(
          <Input.Text value="" onChange={vi.fn()} placeholder="Enter text" />,
        );
        const input = c.getByRole("textbox") as HTMLInputElement;
        expect(input.placeholder).toBe("Enter text");
      });

      it("should render a React node placeholder when value is empty", () => {
        const c = render(
          <Input.Text
            value=""
            onChange={vi.fn()}
            placeholder={<span>Custom placeholder</span>}
          />,
        );
        expect(c.getByText("Custom placeholder")).toBeTruthy();
      });

      it("should not show React node placeholder when value is not empty", () => {
        const c = render(
          <Input.Text
            value="test"
            onChange={vi.fn()}
            placeholder={<span>Custom placeholder</span>}
          />,
        );
        expect(c.queryByText("Custom placeholder")).not.toBeTruthy();
      });
    });

    describe("endContent", () => {
      it("should render end content", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} endContent="units" />);
        expect(c.getByText("units")).toBeTruthy();
      });
    });

    describe("status", () => {
      it("should add status class when status is provided", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} status="error" />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--error");
      });

      it("should add success status class", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} status="success" />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--success");
      });

      it("should add warning status class", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} status="warning" />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--warning");
      });
    });

    describe("keyboard handlers", () => {
      it("should handle keyDown events", () => {
        const onKeyDown = vi.fn();
        const c = render(
          <Input.Text value="" onChange={vi.fn()} onKeyDown={onKeyDown} />,
        );
        const input = c.getByRole("textbox");
        fireEvent.keyDown(input, { key: "Escape" });
        expect(onKeyDown).toHaveBeenCalled();
      });
    });

    describe("custom color", () => {
      it("should apply custom outline color", () => {
        const c = render(<Input.Text value="" onChange={vi.fn()} color="#ff0000" />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto-btn--custom-color");
        expect(container?.style.getPropertyValue("--pluto-btn-color")).toBe(
          "255, 0, 0",
        );
      });
    });
  });

  describe("Numeric", () => {
    describe("Basic Rendering", () => {
      it("should render a numeric input with the provided value", () => {
        const c = render(<Input.Numeric value={42} onChange={vi.fn()} />);
        const input = c.getByRole("textbox");
        expect(input).toBeTruthy();
        expect((input as HTMLInputElement).value).toBe("42");
      });

      it("should render with medium size by default", () => {
        const c = render(<Input.Numeric value={0} onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-medium");
      });

      it("should show drag handle by default", () => {
        const c = render(<Input.Numeric value={0} onChange={vi.fn()} />);
        const dragButton = c.getByLabelText("pluto-icon--drag");
        expect(dragButton).toBeTruthy();
      });
    });

    describe("size", () => {
      it("should render a small numeric input", () => {
        const c = render(<Input.Numeric size="small" value={0} onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-small");
      });

      it("should render a large numeric input", () => {
        const c = render(<Input.Numeric size="large" value={0} onChange={vi.fn()} />);
        const container = c.getByRole("textbox").parentElement;
        expect(container?.className).toContain("pluto--height-large");
      });
    });

    describe("onChange behavior", () => {
      it("should call onChange with parsed number", () => {
        const onChange = vi.fn();
        const c = render(<Input.Numeric value={0} onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "42" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith(42);
      });

      it("should handle mathematical expressions", () => {
        const onChange = vi.fn();
        const c = render(<Input.Numeric value={0} onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "2 + 3" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith(5);
      });

      it("should reset to previous value on invalid input", () => {
        const onChange = vi.fn();
        const c = render(<Input.Numeric value={42} onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "invalid" } });
        fireEvent.blur(input);
        expect((input as HTMLInputElement).value).toBe("42");
        expect(onChange).not.toHaveBeenCalled();
      });

      it("should evaluate on Enter key", () => {
        const onChange = vi.fn();
        const c = render(<Input.Numeric value={0} onChange={onChange} />);
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "10" } });
        fireEvent.keyDown(input, { code: "Enter" });
        expect(onChange).toHaveBeenCalledWith(10);
      });
    });

    describe("bounds", () => {
      it("should clamp values to bounds", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric
            value={5}
            onChange={onChange}
            bounds={{ lower: 0, upper: 10 }}
          />,
        );
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "15" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith(9);
      });

      it("should clamp negative values to lower bound", () => {
        const onChange = vi.fn();
        const c = render(
          <Input.Numeric
            value={5}
            onChange={onChange}
            bounds={{ lower: 0, upper: 10 }}
          />,
        );
        const input = c.getByRole("textbox");
        fireEvent.change(input, { target: { value: "-5" } });
        fireEvent.blur(input);
        expect(onChange).toHaveBeenCalledWith(0);
      });
    });

    describe("disabled", () => {
      it("should not show drag handle when disabled", () => {
        const c = render(<Input.Numeric value={0} onChange={vi.fn()} disabled />);
        const dragButton = c.container.querySelector(".pluto-input-drag-button");
        expect(dragButton).not.toBeTruthy();
      });

      it("should disable the input when disabled is true", () => {
        const c = render(<Input.Numeric value={0} onChange={vi.fn()} disabled />);
        const input = c.getByRole("textbox") as HTMLInputElement;
        expect(input.disabled).toBe(true);
      });
    });

    describe("showDragHandle", () => {
      it("should hide drag handle when showDragHandle is false", () => {
        const c = render(
          <Input.Numeric value={0} onChange={vi.fn()} showDragHandle={false} />,
        );
        const dragButton = c.container.querySelector(".pluto-input-drag-button");
        expect(dragButton).not.toBeTruthy();
      });
    });

    describe("blur handler", () => {
      it("should call onBlur when provided", () => {
        const onBlur = vi.fn();
        const c = render(
          <Input.Numeric value={42} onChange={vi.fn()} onBlur={onBlur} />,
        );
        const input = c.getByRole("textbox");
        fireEvent.blur(input);
        expect(onBlur).toHaveBeenCalled();
      });
    });
  });

  describe("Checkbox", () => {
    describe("Basic Rendering", () => {
      it("should render a checkbox input", () => {
        const c = render(<Input.Checkbox value={false} onChange={vi.fn()} />);
        const checkbox = c.container.querySelector('input[type="checkbox"]');
        expect(checkbox).toBeTruthy();
        expect((checkbox as HTMLInputElement).checked).toBe(false);
      });

      it("should render with medium size by default", () => {
        const c = render(<Input.Checkbox value={false} onChange={vi.fn()} />);
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto--height-medium");
      });
    });

    describe("size", () => {
      it("should render a small checkbox", () => {
        const c = render(
          <Input.Checkbox size="small" value={false} onChange={vi.fn()} />,
        );
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto--height-small");
      });

      it("should render a large checkbox", () => {
        const c = render(
          <Input.Checkbox size="large" value={false} onChange={vi.fn()} />,
        );
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto--height-large");
      });
    });

    describe("value and onChange", () => {
      it("should render checked when value is true", () => {
        const c = render(<Input.Checkbox value onChange={vi.fn()} />);
        const checkbox = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        expect(checkbox.checked).toBe(true);
      });

      it("should call onChange with new boolean value", () => {
        const onChange = vi.fn();
        const c = render(<Input.Checkbox value={false} onChange={onChange} />);
        const checkbox = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        fireEvent.click(checkbox);
        expect(onChange).toHaveBeenCalledWith(true);
      });

      it("should toggle from true to false", () => {
        const onChange = vi.fn();
        const c = render(<Input.Checkbox value onChange={onChange} />);
        const checkbox = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        fireEvent.click(checkbox);
        expect(onChange).toHaveBeenCalledWith(false);
      });
    });

    describe("disabled", () => {
      it("should disable the checkbox when disabled is true", () => {
        const c = render(<Input.Checkbox value={false} onChange={vi.fn()} disabled />);
        const checkbox = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        expect(checkbox.disabled).toBe(true);
      });
    });

    describe("styling", () => {
      it("should apply checkbox CSS classes", () => {
        const c = render(<Input.Checkbox value={false} onChange={vi.fn()} />);
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto-input__checkbox");
      });

      it("should render checkmark element", () => {
        const c = render(<Input.Checkbox value={false} onChange={vi.fn()} />);
        const checkmark = c.container.querySelector(".pluto-input__checkbox-indicator");
        expect(checkmark).toBeTruthy();
      });
    });
  });

  describe("Switch", () => {
    describe("Basic Rendering", () => {
      it("should render a switch input", () => {
        const c = render(<Input.Switch value={false} onChange={vi.fn()} />);
        const switchInput = c.container.querySelector('input[type="checkbox"]');
        expect(switchInput).toBeTruthy();
        expect((switchInput as HTMLInputElement).checked).toBe(false);
      });

      it("should render switch track element", () => {
        const c = render(<Input.Switch value={false} onChange={vi.fn()} />);
        const track = c.container.querySelector(".pluto-input__switch-indicator");
        expect(track).toBeTruthy();
      });
    });

    describe("size", () => {
      it("should render a small switch", () => {
        const c = render(
          <Input.Switch size="small" value={false} onChange={vi.fn()} />,
        );
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto--height-small");
      });

      it("should render a large switch", () => {
        const c = render(
          <Input.Switch size="large" value={false} onChange={vi.fn()} />,
        );
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto--height-large");
      });
    });

    describe("value and onChange", () => {
      it("should render on when value is true", () => {
        const c = render(<Input.Switch value onChange={vi.fn()} />);
        const switchInput = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        expect(switchInput.checked).toBe(true);
      });

      it("should call onChange with new boolean value", () => {
        const onChange = vi.fn();
        const c = render(<Input.Switch value={false} onChange={onChange} />);
        const switchInput = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        fireEvent.click(switchInput);
        expect(onChange).toHaveBeenCalledWith(true);
      });

      it("should toggle from true to false", () => {
        const onChange = vi.fn();
        const c = render(<Input.Switch value onChange={onChange} />);
        const switchInput = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        fireEvent.click(switchInput);
        expect(onChange).toHaveBeenCalledWith(false);
      });
    });

    describe("disabled", () => {
      it("should disable the switch when disabled is true", () => {
        const c = render(<Input.Switch value={false} onChange={vi.fn()} disabled />);
        const switchInput = c.container.querySelector(
          'input[type="checkbox"]',
        ) as HTMLInputElement;
        expect(switchInput.disabled).toBe(true);
      });
    });

    describe("styling", () => {
      it("should apply switch CSS classes", () => {
        const c = render(<Input.Switch value={false} onChange={vi.fn()} />);
        const container = c.container.querySelector("label");
        expect(container?.className).toContain("pluto-input__switch");
      });
    });
  });
});
