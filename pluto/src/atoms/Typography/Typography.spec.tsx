import { render, fireEvent } from "@testing-library/react";
import { AiFillCloseCircle } from "react-icons/ai";
import { describe, expect, it } from "vitest";

import { Text } from ".";

describe("Text", () => {
  describe("Basic", () => {
    it("should render text with the correct HTML tag", () => {
      const c = render(<Text level="h2">Hello</Text>);
      expect(c.getByText("Hello").tagName).toBe("H2");
    });
  });
  describe("WithIcon", () => {
    it("should render text with a starting icon", () => {
      const c = render(
        <Text.WithIcon startIcon={<AiFillCloseCircle aria-label="close" />} level="h2">
          Hello
        </Text.WithIcon>
      );
      expect(c.getByLabelText("close")).toBeTruthy();
    });
    it("should render text with an ending icon", () => {
      const c = render(
        <Text.WithIcon endIcon={<AiFillCloseCircle aria-label="close" />} level="h2">
          Hello
        </Text.WithIcon>
      );
      expect(c.getByLabelText("close")).toBeTruthy();
    });
  });
  describe("Editable", () => {
    it("should focus and select the text when double clicked", () => {
      const c = render(<Text.Editable level="h1">Hello</Text.Editable>);
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(document.activeElement).toBe(text);
      expect(window.getSelection()?.toString()).toBe("Hello");
    });
  });
});
