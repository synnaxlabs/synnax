import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Generic } from "@/generic";

describe("Generic", () => {
  it("should render a div", () => {
    const { container } = render(<Generic.Element el="input" />);
    expect(container.firstChild).toBeInstanceOf(HTMLDivElement);
  });
});
