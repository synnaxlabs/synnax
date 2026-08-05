// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { GROUP } from "@/schematic/node/general/group";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { StringDisplayForm } from "@/schematic/node/general/stringDisplay/Form";
import { StringDisplay as Primitive } from "@/schematic/node/general/stringDisplay/Primitive";
import { telem } from "@/telem/aether";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Theming } from "@/theming";

const theme = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark);

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof StringDisplay.configZ>({
    values: deep.copy(StringDisplay.defaultConfig(theme)),
    schema: StringDisplay.configZ,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof StringDisplay.configZ> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

const getText = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-text");
  if (el == null) throw new Error("expected a text element");
  return el;
};

describe("StringDisplay", () => {
  describe("defaultConfig", () => {
    it("should produce a config that satisfies its own schema", () => {
      const config = StringDisplay.defaultConfig(theme);
      expect(StringDisplay.configZ.parse(config)).toEqual(config);
    });

    it("should source telemetry from a bare string source, not a pipeline", () => {
      const { telem: t } = StringDisplay.defaultConfig(theme);
      expect(t?.variant).toBe("source");
      expect(t?.valueType).toBe("string");
      expect(t?.type).toBe("stream-channel-string-value");
    });
  });

  describe("configZ", () => {
    it("should reject a telem spec that emits numbers", () => {
      const config = {
        ...StringDisplay.defaultConfig(theme),
        telem: telem.streamChannelValue({ channel: 1 }),
      };
      const { success, error } = StringDisplay.configZ.safeParse(config);
      expect(success).toBe(false);
      expect(error).toBeInstanceOf(z.ZodError);
      expect(error?.issues).toHaveLength(1);
      expect(error?.issues[0].path).toEqual(["telem", "valueType"]);
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
          textColor="#ffffff"
          stalenessColor={color.construct("#ff0000")}
        />,
      );
      expect(getText(container).style.color).toBe("rgb(255, 255, 255)");
    });

    it("should use the staleness color once the value goes stale", () => {
      const { container } = render(
        <Primitive
          value="hello"
          textColor="#ffffff"
          stalenessColor={color.construct("#ff0000")}
          stale
        />,
      );
      expect(getText(container).style.color).toBe("rgb(255, 0, 0)");
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
      expect(getByText("Input channel")).toBeDefined();
      expect(getByText("Stale color")).toBeDefined();
      expect(getByText("Stale timeout")).toBeDefined();
    });
  });

  // Console's StaticSymbolList filters by group, so a symbol missing from
  // GROUP.symbols is reachable only through search.
  it("should be listed in the general group", () => {
    expect(GROUP.symbols).toContain("stringDisplay");
  });
});
