// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TankDiagram } from "@/components/automate/diagrams/TankDiagram";
import { ZERO_DIAGRAM_STATE } from "@/components/automate/timeline";

const EMERGENCY = "var(--pluto-error-z)";

const fill = (svg: HTMLElement): SVGRectElement =>
  svg.querySelector("rect[clip-path]") as SVGRectElement;

describe("TankDiagram", () => {
  describe("pressure", () => {
    it("should show the pressure to one decimal", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, pressure: 150.25 }}
          fullScalePressure={600}
        />,
      );
      expect(container.textContent).toContain("150.3 psi");
    });

    it("should fill the tank in proportion to the full-scale pressure", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, pressure: 300 }}
          fullScalePressure={600}
        />,
      );
      expect(fill(container).getAttribute("height")).toBe("40");
    });

    it("should cap the fill at the top of the tank", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, pressure: 1200 }}
          fullScalePressure={600}
        />,
      );
      expect(fill(container).getAttribute("height")).toBe("80");
    });

    it("should give each instance its own clip path", () => {
      const { container } = render(
        <>
          <TankDiagram state={ZERO_DIAGRAM_STATE} fullScalePressure={600} />
          <TankDiagram state={ZERO_DIAGRAM_STATE} fullScalePressure={900} />
        </>,
      );
      const ids = [...container.querySelectorAll("clipPath")].map((c) => c.id);
      expect(new Set(ids).size).toBe(2);
      const refs = [...container.querySelectorAll("rect[clip-path]")].map((r) =>
        r.getAttribute("clip-path"),
      );
      expect(refs).toEqual(ids.map((id) => `url(#${id})`));
    });
  });

  describe("authority", () => {
    it("should hide the authority meter by default", () => {
      const { container } = render(
        <TankDiagram state={ZERO_DIAGRAM_STATE} fullScalePressure={600} />,
      );
      expect(container.textContent).not.toContain("auth");
    });

    it("should show the authority meter when asked", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, authority: 255 }}
          fullScalePressure={600}
          authorityVisible
        />,
      );
      expect(container.textContent).toContain("auth 255");
    });
  });

  describe("stage", () => {
    it("should show the stage name", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, stage: "pressurize" }}
          fullScalePressure={600}
        />,
      );
      expect(container.textContent).toContain("pressurize");
      expect(container.textContent).not.toContain("SAFE");
    });

    it("should show the safe badge when safed", () => {
      const { container } = render(
        <TankDiagram
          state={{ ...ZERO_DIAGRAM_STATE, stage: "safed" }}
          fullScalePressure={600}
        />,
      );
      expect(container.textContent).toContain("SAFE");
    });

    it("should color the tank and the vent in the emergency color", () => {
      const { container } = render(
        <TankDiagram
          state={{
            ...ZERO_DIAGRAM_STATE,
            stage: "emergency",
            ventValve: true,
            pressure: 100,
          }}
          fullScalePressure={600}
        />,
      );
      expect(fill(container).getAttribute("fill")).toBe(EMERGENCY);
      const strokes = [...container.querySelectorAll("path, line")].map((el) =>
        el.getAttribute("stroke"),
      );
      expect(strokes).toContain(EMERGENCY);
    });
  });
});
