// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/Diagram.css";
import "@xyflow/react/dist/base.css";

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";

const renderDiagram = async (colorMode: "light" | "dark"): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <div className={CSS.BE("diagram", "container")}>
      <div className={CSS.cls("react-flow", colorMode, CSS.B("diagram"))} />
    </div>,
  );
  const diagram = container.querySelector<HTMLElement>(".react-flow");
  if (diagram == null) throw new Error("expected a diagram element");
  return diagram;
};

describe("Diagram style", () => {
  it.each(["light", "dark"] as const)(
    "should show the theme through the background in the %s color mode",
    async (colorMode) => {
      const diagram = await renderDiagram(colorMode);
      expect(getComputedStyle(diagram).backgroundColor).toBe("rgba(0, 0, 0, 0)");
    },
  );
});
