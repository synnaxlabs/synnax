import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Resize } from ".";

// Testing the Resize is extremely difficult on a unit level
// using jsdom, so we're just going to test that it renders
// for now.
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
