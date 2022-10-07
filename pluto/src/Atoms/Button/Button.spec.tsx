import { expect, describe, it } from "vitest";
import { render } from "@testing-library/react";
import Button from "./Button";

describe("Button", () => {
  it("should render a button with the provided text", () => {
    const c = render(<Button size="small">Hello</Button>);
    expect(c.getByText("Hello")).toBeTruthy();
  });
});
