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
