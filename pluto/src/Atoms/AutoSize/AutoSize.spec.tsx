import { expect, describe, it, afterAll, vi, beforeAll } from "vitest";
import { render } from "@testing-library/react";
import { AutoSize } from ".";

describe("AutoSize", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = vi.fn(
      (): DOMRect => ({
        width: 100,
        height: 200,
        top: 0,
        left: 0,
        bottom: 0,
        right: 0,
        x: 0,
        y: 0,
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        toJSON: () => {},
      })
    );
  });
  afterAll(() => {
    vi.clearAllMocks();
  });
  it("should provide a width and height to a child component", async () => {
    const c = render(
      <div style={{ height: 100, width: 100 }}>
        <AutoSize style={{ height: 100, width: 100 }}>
          {({ width, height }) => {
            return (
              <div className="hello" style={{ height, width }}>
                Hello
              </div>
            );
          }}
        </AutoSize>
      </div>
    );

    expect(c.getByText("Hello").style.width).toBe("100px");
  });
  it("should provide a width and height to a child element", async () => {
    const El = ({ width, height }: { width?: number; height?: number }) => {
      return <div style={{ height, width }}>Hello</div>;
    };
    const c = render(
      <div style={{ height: 100, width: 100 }}>
        <AutoSize style={{ height: 100, width: 100 }}>
          <El />
        </AutoSize>
      </div>
    );

    expect(c.getByText("Hello").style.width).toBe("100px");
  });
});
