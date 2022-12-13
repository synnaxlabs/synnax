import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Header } from ".";

describe("Header", () => {
  it("should render a header", () => {
    const c = render(<Header level="h1">Header</Header>);
    expect(c.getByText("Header")).toBeTruthy();
  });
});