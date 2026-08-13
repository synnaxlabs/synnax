// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { asyncEdit, edit, Editable, MaybeEditable } from "@/text/Editable";

describe("Editable", () => {
  describe("rendering", () => {
    it("should render the value as text", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      expect(c.getByText("Hello")).toBeDefined();
    });

    it("should not be in edit mode initially", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      expect(text.getAttribute("contenteditable")).toBe("false");
    });

    it("should forward refs to the underlying element", () => {
      const ref = createRef<HTMLParagraphElement>();
      const c = render(<Editable ref={ref} value="Hello" onChange={vi.fn()} />);
      expect(ref.current).not.toBeNull();
      expect(ref.current).toBe(c.getByText("Hello"));
    });
  });

  describe("entering edit mode", () => {
    it("should enter edit mode on double-click", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(text.getAttribute("contenteditable")).toBe("true");
    });

    it("should not enter edit mode on double-click when allowDoubleClick is false", () => {
      const c = render(
        <Editable value="Hello" onChange={vi.fn()} allowDoubleClick={false} />,
      );
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(text.getAttribute("contenteditable")).toBe("false");
    });

    it("should still invoke onDoubleClick when allowDoubleClick is false", () => {
      const onDoubleClick = vi.fn();
      const c = render(
        <Editable
          value="Hello"
          onChange={vi.fn()}
          allowDoubleClick={false}
          onDoubleClick={onDoubleClick}
        />,
      );
      fireEvent.dblClick(c.getByText("Hello"));
      expect(onDoubleClick).toHaveBeenCalledTimes(1);
    });
  });

  describe("committing", () => {
    it("should call onChange with the new value when Enter is pressed", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith("World");
    });

    it("should call onChange with the new value when the element is blurred", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "Goodbye";
      fireEvent.blur(text);
      expect(onChange).toHaveBeenCalledWith("Goodbye");
    });

    it("should not call onChange when committing the same value", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "Hello";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should exit edit mode after Enter is pressed", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(text.getAttribute("contenteditable")).toBe("true");
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(text.getAttribute("contenteditable")).toBe("false");
    });
  });

  describe("escaping", () => {
    it("should not call onChange when Escape is pressed", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Escape" });
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should revert the text when Escape is pressed", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Escape" });
      expect(text.innerText).toBe("Hello");
    });
  });

  describe("edit", () => {
    it("rejects asyncEdit when the element never appears", async () => {
      vi.useFakeTimers();
      try {
        const promise = asyncEdit("does-not-exist");
        const expectation = expect(promise).rejects.toThrow(
          "Could not find element",
        );
        await vi.advanceTimersByTimeAsync(1100);
        await expectation;
      } finally {
        vi.useRealTimers();
      }
    });

    it("reports a missing element through console.error instead of throwing", async () => {
      vi.useFakeTimers();
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      try {
        edit("does-not-exist");
        await vi.advanceTimersByTimeAsync(1100);
        expect(consoleError).toHaveBeenCalledTimes(1);
      } finally {
        consoleError.mockRestore();
        vi.useRealTimers();
      }
    });

    it("resolves asyncEdit with the committed value", async () => {
      const c = render(<Editable id="rename-me" value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      let promise!: Promise<[string, boolean]>;
      act(() => {
        promise = asyncEdit("rename-me");
      });
      expect(text.getAttribute("contenteditable")).toBe("true");
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      await expect(promise).resolves.toEqual(["World", true]);
    });

    it("resolves asyncEdit with renamed=false on escape", async () => {
      const c = render(<Editable id="escape-me" value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      let promise!: Promise<[string, boolean]>;
      act(() => {
        promise = asyncEdit("escape-me");
      });
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Escape" });
      await expect(promise).resolves.toEqual(["Hello", false]);
    });

    it("should put the element with the given id into edit mode", () => {
      const c = render(<Editable id="tab-name" value="Hello" onChange={vi.fn()} />);
      edit("tab-name");
      expect(c.getByText("Hello").getAttribute("contenteditable")).toBe("true");
    });

    it("should edit the editable copy when a read-only copy shares its id", () => {
      const c = render(
        <>
          <MaybeEditable
            id="tab-name"
            value="Hello"
            onChange={vi.fn<(value: string) => void>()}
            disabled
          />
          <Editable id="tab-name" value="Hello" onChange={vi.fn()} />
        </>,
      );
      edit("tab-name");
      const [readOnly, editable] = c.getAllByText("Hello");
      expect(editable.getAttribute("contenteditable")).toBe("true");
      expect(readOnly.getAttribute("contenteditable")).toBeNull();
    });
  });

  describe("empty values", () => {
    it("should reject empty values by default", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).not.toHaveBeenCalled();
      expect(text.innerText).toBe("Hello");
    });

    it("should accept empty values when allowEmpty is true", () => {
      const onChange = vi.fn();
      const c = render(<Editable value="Hello" onChange={onChange} allowEmpty />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).toHaveBeenCalledWith("");
    });
  });
});
