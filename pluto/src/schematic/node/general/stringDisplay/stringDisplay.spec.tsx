// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { color, deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Form } from "@/form";
import { GROUP } from "@/schematic/node/general/group";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { StringDisplayForm } from "@/schematic/node/general/stringDisplay/Form";
import { StringDisplay as Primitive } from "@/schematic/node/general/stringDisplay/Primitive";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { SYNNAX_DARK, type Theme, themeZ } from "@/theming/base/theme";

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const THEME: Theme = themeZ.parse(SYNNAX_DARK);
const CONFIG_Z = schematic.stringDisplayNodeConfigZ;

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof CONFIG_Z>({
    values: deep.copy(StringDisplay.defaultConfig(THEME)),
    schema: CONFIG_Z,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

const getText = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-text");
  if (el == null) throw new Error("expected a text element");
  return el;
};

const getBox = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(`.${CSS.B("string-display")}`);
  if (el == null) throw new Error("expected a string display element");
  return el;
};

const LONG_VALUE = "a".repeat(500);

describe("StringDisplay", () => {
  describe("defaultConfig", () => {
    it("should produce a config that satisfies its own schema", () => {
      const config = StringDisplay.defaultConfig(THEME);
      expect(CONFIG_Z.parse(config)).toEqual(config);
    });
  });

  describe("Primitive", () => {
    it("should render the value as text", () => {
      const { container } = render(<Primitive value="hello" />);
      expect(getText(container).textContent).toBe("hello");
    });

    // The box takes its height from the text element's line box, so the element has to
    // be in the DOM before a value arrives or the symbol renders collapsed.
    it("should render a text element when no value has arrived", () => {
      const { container } = render(<Primitive value="" />);
      expect(getText(container).textContent).toBe("");
    });

    it("should use the text color while the value is fresh", () => {
      const { container } = render(
        <Primitive
          value="hello"
          textColor={color.construct("#ffffff")}
          stalenessColor={color.construct("#ff0000")}
        />,
      );
      expect(getText(container).style.color).toBe("rgb(255, 255, 255)");
    });

    it("should use the staleness color once the value goes stale", () => {
      const { container } = render(
        <Primitive
          value="hello"
          textColor={color.construct("#ffffff")}
          stalenessColor={color.construct("#ff0000")}
          stale
        />,
      );
      expect(getText(container).style.color).toBe("rgb(255, 0, 0)");
    });

    // inlineSize is a width, not a floor. If it regresses to a minimum the box grows
    // with the value and the configured width stops being respected.
    it("should keep the configured width no matter how long the value is", () => {
      const { container } = render(<Primitive value={LONG_VALUE} inlineSize={100} />);
      expect(getBox(container).style.width).toBe("100px");
    });

    it("should truncate a value too long for the configured width", () => {
      const { container } = render(<Primitive value={LONG_VALUE} inlineSize={100} />);
      expect(getText(container).className).toContain(
        CSS.BM("text", "overflow", "ellipsis"),
      );
    });
  });

  describe("Form", () => {
    const renderForm = (): ReturnType<typeof render> =>
      render(
        <FormWrapper>
          <StringDisplayForm />
        </FormWrapper>,
      );

    it("should render the style controls", () => {
      const { getByText } = renderForm();
      expect(getByText("Label")).toBeDefined();
      expect(getByText("Color")).toBeDefined();
      expect(getByText("Display width")).toBeDefined();
      expect(getByText("Size")).toBeDefined();
    });

    it("should render the telemetry controls", () => {
      const { getByText } = renderForm();
      fireEvent.click(getByText("Telemetry"));
      expect(getByText("Channel")).toBeDefined();
      expect(getByText("Stale color")).toBeDefined();
      expect(getByText("Stale timeout")).toBeDefined();
    });
  });

  // Console's StaticSymbolList filters by group, so a symbol missing from
  // GROUP.symbols is reachable only through search.
  it("should be listed in the general group", () => {
    expect(GROUP.symbols).toContain("string_display");
  });
});
