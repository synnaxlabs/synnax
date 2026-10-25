// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { color } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { LightForm } from "@/schematic/node/general/light/Form";
import { Light } from "@/schematic/node/general/light/Primitive";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const CONFIG_Z = schematic.lightNodeConfigZ;

const COLOR_VAR = "--pluto-symbol-color";

const getLamp = (container: HTMLElement): SVGGElement => {
  const el = container.querySelector<SVGGElement>(".pluto-light__lamp");
  if (el == null) throw new Error("expected a lamp group");
  return el;
};

const getSVG = (container: HTMLElement): SVGSVGElement => {
  const el = container.querySelector<SVGSVGElement>("svg");
  if (el == null) throw new Error("expected a light svg");
  return el;
};

describe("light symbol", () => {
  describe("colors", () => {
    // The outline and the lit fill paint from display vars in light.css; jsdom cannot
    // compute them, so we assert the source var each part reads.
    it("should paint the outline from the stroke and the lamp from the on color", () => {
      const { container } = render(
        <Light
          strokeColor={color.construct("#00ff00")}
          onColor={color.construct("#ff0000")}
        />,
      );
      expect(getSVG(container).style.getPropertyValue(COLOR_VAR)).toBe("0, 255, 0, 1");
      expect(getLamp(container).style.getPropertyValue(COLOR_VAR)).toBe("255, 0, 0, 1");
    });

    it("should let the lamp inherit the stroke when the on color is absent", () => {
      const { container } = render(<Light strokeColor={color.construct("#00ff00")} />);
      expect(getLamp(container).style.getPropertyValue(COLOR_VAR)).toBe("");
    });

    it("should mark the lamp as a colored part", () => {
      const { container } = render(<Light onColor={color.construct("#ff0000")} />);
      expect(getLamp(container).getAttribute("class")).toContain(
        "pluto-symbol-colored",
      );
    });
  });

  describe("form", () => {
    const FormWrapper = ({
      children,
      strokeColor,
    }: PropsWithChildren<{ strokeColor?: color.Color }>): ReactElement => {
      const methods = Form.use<typeof CONFIG_Z>({
        values: { ...Node.createConfig({ variant: "light" }), strokeColor },
        schema: CONFIG_Z,
      });
      return (
        <SynnaxWrapper>
          <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
        </SynnaxWrapper>
      );
    };

    const renderStyle = (strokeColor?: color.Color) => {
      const result = render(
        <FormWrapper strokeColor={strokeColor}>
          <LightForm />
        </FormWrapper>,
      );
      fireEvent.click(result.getByText("Style"));
      return result;
    };

    it("should render the stroke and on color fields", () => {
      const { getByText } = renderStyle();
      expect(getByText("Stroke")).toBeDefined();
      expect(getByText("On")).toBeDefined();
    });

    it("should show the stroke as the on color's auto fallback", () => {
      const { container } = renderStyle(color.construct("#00ff00"));
      const swatches = container.querySelectorAll<HTMLElement>(
        ".pluto-color-swatch.pluto--auto",
      );
      expect(swatches).toHaveLength(1);
      expect(swatches[0].style.getPropertyValue("--pluto-swatch-color")).toBe(
        "rgba(0, 255, 0, 1)",
      );
    });
  });
});
