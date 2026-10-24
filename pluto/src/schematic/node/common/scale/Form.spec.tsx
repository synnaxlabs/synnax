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
import { type ReactElement } from "react";
import { assert, describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { ScaleForm } from "@/schematic/node/general/scale/Form";
import { TankForm } from "@/schematic/node/vessels/tank/Form";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const CONFIG_Z = schematic.nodeConfigZ;

interface FormWrapperProps {
  values: schematic.NodeConfig;
  children: ReactElement;
}

const FormWrapper = ({ values, children }: FormWrapperProps): ReactElement => {
  const methods = Form.use<typeof CONFIG_Z>({ values, schema: CONFIG_Z });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

const checked = (el: HTMLElement): boolean => {
  assert(el instanceof HTMLInputElement);
  return el.checked;
};

// The color the swatch of the field with the label shows.
const swatchColor = (container: HTMLElement, label: string): string => {
  const item = Array.from(container.querySelectorAll(".pluto-input__item")).find(
    (el) => el.querySelector("label")?.textContent?.trim() === label,
  );
  const swatch = item?.querySelector<HTMLElement>(".pluto-color-swatch");
  assert(swatch != null, `no color field labeled ${label}`);
  return swatch.style.getPropertyValue("--pluto-swatch-color");
};

describe("scale display fields", () => {
  it("should show the scale's value and axis unless they are hidden", () => {
    const { getByRole, getByText } = render(
      <FormWrapper
        values={{ ...Node.createConfig({ variant: "scale" }), caretHidden: true }}
      >
        <ScaleForm />
      </FormWrapper>,
    );
    fireEvent.click(getByText("Style"));
    expect(checked(getByRole("checkbox", { name: "Value" }))).toBe(false);
    expect(checked(getByRole("checkbox", { name: "Scale" }))).toBe(true);
  });

  it("should hide the tank's value and axis unless they are visible", () => {
    const { getByRole } = render(
      <FormWrapper
        values={{
          ...Node.createConfig({ variant: "tank" }),
          channel: 1,
          caretVisible: true,
        }}
      >
        <TankForm showFillTab />
      </FormWrapper>,
    );
    fireEvent.click(getByRole("tab", { name: "Fill" }));
    expect(checked(getByRole("checkbox", { name: "Value" }))).toBe(true);
    expect(checked(getByRole("checkbox", { name: "Scale" }))).toBe(false);
  });
});

describe("tank colors", () => {
  it("should edit the wall and the axis as separate colors", () => {
    const { container, getByRole } = render(
      <FormWrapper
        values={{
          ...Node.createConfig({ variant: "tank" }),
          channel: 1,
          strokeColor: color.construct("#ff0000"),
          axisColor: color.construct("#00ff00"),
        }}
      >
        <TankForm showFillTab />
      </FormWrapper>,
    );
    expect(swatchColor(container, "Stroke")).toBe("rgba(255, 0, 0, 1)");
    fireEvent.click(getByRole("tab", { name: "Fill" }));
    expect(swatchColor(container, "Axis")).toBe("rgba(0, 255, 0, 1)");
  });
});
