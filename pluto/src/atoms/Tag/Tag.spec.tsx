import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Tag } from ".";

describe("Tag", () => {
  it("should render a tag", () => {
    const c = render(<Tag>Test</Tag>);
    expect(c.getByText("Test")).toBeTruthy();
  });
  it("should render a tag wiht an icon", () => {
    const c = render(<Tag icon={<div>Icon</div>}>Test</Tag>);
    expect(c.getByText("Test")).toBeTruthy();
    expect(c.getByText("Icon")).toBeTruthy();
  });
  it("should render a close button if onClose is provided", () => {
    const close = vi.fn();
    const c = render(<Tag onClose={close}>Test</Tag>);
    const btn = c.getByLabelText("close");
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(close).toHaveBeenCalled();
  });
});
