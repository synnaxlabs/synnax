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

    it("should render markup in the value as literal text", () => {
      const value = "<img src=x onerror=alert(1)>";
      const c = render(<Editable value={value} onChange={vi.fn()} />);
      const text = c.getByText(value);
      expect(text.textContent).toBe(value);
      expect(text.querySelector("img")).toBeNull();
    });

    it("should render committed markup as literal text", () => {
      const value = "<img src=x onerror=alert(1)>";
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = value;
      fireEvent.keyDown(text, { key: "Enter" });
      c.rerender(<Editable value="Hello" onChange={vi.fn()} />);
      expect(text.textContent).toBe(value);
      expect(text.querySelector("img")).toBeNull();
    });

    it("should not render a one-character value as a square", () => {
      const c = render(<Editable value="a" onChange={vi.fn()} />);
      expect(c.getByText("a").className).not.toContain("lyra--square");
    });

    it("should not render a read-only one-character value as a square", () => {
      const c = render(<MaybeEditable value="a" />);
      expect(c.getByText("a").className).not.toContain("lyra--square");
    });

    it("should forward refs to the underlying element", () => {
      const ref = createRef<HTMLParagraphElement>();
      const c = render(<Editable ref={ref} value="Hello" onChange={vi.fn()} />);
      expect(ref.current).not.toBeNull();
      expect(ref.current).toBe(c.getByText("Hello"));
    });
  });

  describe("maxLength", () => {
    const renderField = (maxLength?: number): HTMLElement => {
      const c = render(
        <Editable value="abc" onChange={vi.fn()} maxLength={maxLength} />,
      );
      const text = c.getByText("abc");
      fireEvent.dblClick(text);
      return text;
    };

    it("should drop characters typed past the limit", () => {
      const text = renderField(3);
      text.textContent = "abcd";
      window.getSelection()?.collapse(text.firstChild, 4);
      fireEvent.input(text);
      expect(text.textContent).toBe("abc");
    });

    it("should drop only the characters inserted before the caret", () => {
      const text = renderField(3);
      text.textContent = "aXYbc";
      window.getSelection()?.collapse(text.firstChild, 3);
      fireEvent.input(text);
      expect(text.textContent).toBe("abc");
      expect(window.getSelection()?.focusOffset).toBe(1);
    });

    it("should find the caret inside pasted markup", () => {
      const text = renderField(3);
      text.innerHTML = "a<b>XY</b>bc";
      window.getSelection()?.collapse(text.querySelector("b")?.firstChild ?? null, 2);
      fireEvent.input(text);
      expect(text.textContent).toBe("abc");
    });

    it("should drop a whole emoji rather than half of one", () => {
      const text = renderField(3);
      text.textContent = "abc🚀";
      window.getSelection()?.collapse(text.firstChild, 5);
      fireEvent.input(text);
      expect(text.textContent).toBe("abc");
    });

    it("should keep the start of text pasted over the limit", () => {
      const text = renderField(3);
      text.textContent = "abcdefgh";
      window.getSelection()?.collapse(text.firstChild, 8);
      fireEvent.input(text);
      expect(text.textContent).toBe("abc");
    });

    it("should leave text alone without a limit", () => {
      const text = renderField();
      text.textContent = "abcdefgh";
      fireEvent.input(text);
      expect(text.textContent).toBe("abcdefgh");
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

  describe("repainting after a commit", () => {
    it("should keep the committed text while the value prop is stale", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(text.innerHTML).toBe("World");
      c.rerender(<Editable value="Hello" onChange={vi.fn()} />);
      expect(text.innerHTML).toBe("World");
    });

    it("should adopt the value once the write lands", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      c.rerender(<Editable value="World" onChange={vi.fn()} />);
      expect(text.innerHTML).toBe("World");
    });

    it("should show the value again when the write rolls back", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      c.rerender(<Editable value="World" onChange={vi.fn()} />);
      c.rerender(<Editable value="Hello" onChange={vi.fn()} />);
      expect(text.innerHTML).toBe("Hello");
    });

    it("should repaint a remote rename that beats the commit", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      c.rerender(<Editable value="Elsewhere" onChange={vi.fn()} />);
      expect(text.innerHTML).toBe("Elsewhere");
    });
    it("should revert when the handler rejects the committed text", () => {
      const onChange = vi.fn<(value: string) => boolean>(() => false);
      const c = render(<Editable value="Hello" onChange={onChange} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "Taken";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).toHaveBeenCalledWith("Taken");
      expect(text.innerText).toBe("Hello");
      c.rerender(<Editable value="Hello" onChange={onChange} />);
      expect(text.innerHTML).toBe("Hello");
    });

    it("should resolve asyncEdit as a cancel when the handler rejects", async () => {
      const c = render(<Editable id="rejected" value="Hello" onChange={() => false} />);
      const text = c.getByText("Hello");
      let promise!: Promise<[string, boolean]>;
      act(() => {
        promise = asyncEdit("rejected");
      });
      text.innerText = "Taken";
      fireEvent.keyDown(text, { key: "Enter" });
      await expect(promise).resolves.toEqual(["Hello", false]);
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

  describe("focus", () => {
    const renderWithList = () => {
      const c = render(
        <div tabIndex={0} data-testid="list">
          <Editable value="Hello" onChange={vi.fn()} />
        </div>,
      );
      const list = c.getByTestId("list");
      list.focus();
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(document.activeElement).toBe(text);
      return { list, text };
    };

    it("should give the focus back when Enter commits", () => {
      const { list, text } = renderWithList();
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(document.activeElement).toBe(list);
    });

    it("should give the focus back when Escape reverts", () => {
      const { list, text } = renderWithList();
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Escape" });
      expect(document.activeElement).toBe(list);
    });

    it("should blur when nothing held the focus before the edit", () => {
      const c = render(<Editable value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(document.activeElement).toBe(document.body);
    });
  });

  describe("edit", () => {
    it("rejects asyncEdit when the element never appears", async () => {
      vi.useFakeTimers();
      try {
        const promise = asyncEdit("does-not-exist");
        const expectation = expect(promise).rejects.toThrow("Could not find element");
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

    it("should start an edit after a level change renders a new element", async () => {
      const onChange = vi.fn();
      const c = render(
        <Editable id="relevel" level="p" value="Hello" onChange={onChange} />,
      );
      c.rerender(
        <Editable id="relevel" level="h3" value="Hello" onChange={onChange} />,
      );
      const text = c.getByText("Hello");
      expect(text.tagName).toBe("H3");
      let promise!: Promise<[string, boolean]>;
      act(() => {
        promise = asyncEdit("relevel");
      });
      text.innerText = "World";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onChange).toHaveBeenCalledWith("World");
      await expect(promise).resolves.toEqual(["World", true]);
    });

    it("resolves asyncEdit with renamed=false when the value is unchanged", async () => {
      const c = render(<Editable id="same-name" value="Hello" onChange={vi.fn()} />);
      const text = c.getByText("Hello");
      let promise!: Promise<[string, boolean]>;
      act(() => {
        promise = asyncEdit("same-name");
      });
      text.innerText = "Hello";
      fireEvent.keyDown(text, { key: "Enter" });
      await expect(promise).resolves.toEqual(["Hello", false]);
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
      act(() => edit("tab-name"));
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
      act(() => edit("tab-name"));
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
