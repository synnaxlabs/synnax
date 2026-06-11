// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { Base } from "@/schematic/edge/common/base";

const renderEdge = (node: ReactNode) => render(<svg>{node}</svg>);

const queryPath = (container: HTMLElement): SVGPathElement => {
  const el = container.querySelector<SVGPathElement>("path.react-flow__edge-path");
  if (el == null) throw new Error("expected an edge path element");
  return el;
};

describe("edge Base", () => {
  it("should set the symbol-color variable and marker class for an explicit color", () => {
    const { container } = renderEdge(<Base.Base path="M0 0 L10 10" color="#ff0000" />);
    const path = queryPath(container);
    expect(path.style.getPropertyValue("--pluto-symbol-color")).toMatch(
      /255\s*,\s*0\s*,\s*0/,
    );
    expect(path.getAttribute("class")).toContain("pluto-symbol-colored");
  });

  it("should treat the ZERO sentinel as unset so it falls back to the theme", () => {
    const { container } = renderEdge(
      <Base.Base path="M0 0 L10 10" color={color.ZERO} />,
    );
    expect(queryPath(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "",
    );
  });

  it("should carry the alpha channel so a translucent edge stays translucent", () => {
    const { container } = renderEdge(
      <Base.Base path="M0 0 L10 10" color={[255, 0, 0, 0.5]} />,
    );
    expect(queryPath(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "255, 0, 0, 0.5",
    );
  });

  it("should stroke a CSS variable string directly and skip the transform", () => {
    const { container } = renderEdge(
      <Base.Base path="M0 0 L10 10" color="var(--pluto-error-z)" />,
    );
    const path = queryPath(container);
    // The preview's CSS-var color is not parseable, so no source var is set.
    expect(path.style.getPropertyValue("--pluto-symbol-color")).toBe("");
    expect(path.style.stroke).toBe("var(--pluto-error-z)");
  });
});
