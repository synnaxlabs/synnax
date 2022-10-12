import { expect, describe, it } from "vitest";
import { render } from "@testing-library/react";
import { Resize } from ".";

describe("Resize", () => {
  describe("Resize", () => {
    it("should render a resize div", async () => {
      const c = render(
        <Resize location="left" initialSize={50} minSize={20} maxSize={500}>
          <p>Hello</p>
        </Resize>
      );
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });
});
