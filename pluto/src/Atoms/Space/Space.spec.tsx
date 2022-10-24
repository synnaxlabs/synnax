import { expect, describe, it } from "vitest";
import { render } from "@testing-library/react";
import Space from "./Space";

describe("Space", () => {
  it("should render items with a space between them", () => {
    const c = render(
      <Space size="small">
        <div>Hello</div>
        <div>World</div>
      </Space>
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    const parent = world.parentElement;
    expect(parent?.classList.toString()).toContain("small");
  });
  it("should render items with no gap", () => {
    const c = render(
      <Space empty>
        <div>Hello</div>
        <div>World</div>
      </Space>
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    const parent = world.parentElement;
    expect(parent?.style.gap).toBe("0");
  });
  it("should render items with a multiple of the base size", () => {
    const c = render(
      <Space size={2}>
        <div>Hello</div>
        <div>World</div>
      </Space>
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    expect(world.parentElement?.style.gap).toBe("2rem");
  });
});
