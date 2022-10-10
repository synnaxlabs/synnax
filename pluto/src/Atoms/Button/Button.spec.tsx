import { expect, describe, it } from "vitest";
import { render } from "@testing-library/react";
import Button from ".";
import { AiOutlineAim } from "react-icons/ai";

describe("Button", () => {
  describe("Default", () => {
    it("should render a button with the provided text", () => {
      const c = render(<Button size="small">Hello</Button>);
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });
  describe("IconOnly", () => {
    it("should render a butto with the provided icon", () => {
      const c = render(
        <Button.IconOnly size="small">
          <AiOutlineAim aria-label="icon" />
        </Button.IconOnly>
      );
      expect(c.getByLabelText("icon")).toBeTruthy();
    });
  });
});
