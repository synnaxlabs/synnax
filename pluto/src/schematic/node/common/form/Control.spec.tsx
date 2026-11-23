// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Control } from "@/schematic/node/common/control";
import { ControlChipField } from "@/schematic/node/common/form/Control";

const schema = z.object({ control: Control.stateConfigZ });

const path = deep.path<z.infer<typeof schema>>();

const Stored = (): ReactElement => {
  const control = Form.useFieldValue(path.control);
  return <div data-testid="stored">{JSON.stringify(control)}</div>;
};

const renderField = (control: Control.StateConfig) => {
  const Harness = (): ReactElement => {
    const methods = Form.use({ values: { control }, schema });
    return (
      <Form.Form<typeof schema> {...methods}>
        <ControlChipField />
        <Stored />
      </Form.Form>
    );
  };
  return render(<Harness />);
};

const stored = (c: ReturnType<typeof render>): Control.StateConfig =>
  Control.stateConfigZ.parse(JSON.parse(c.getByTestId("stored").textContent ?? ""));

describe("ControlChipField", () => {
  it("should read as on for the default control state", () => {
    const c = renderField(Control.stateConfigZ.parse({}));
    expect(c.getByRole("checkbox")).toHaveProperty("checked", true);
  });

  it("should hide the default control state when turned off", () => {
    const c = renderField(Control.stateConfigZ.parse({}));
    fireEvent.click(c.getByRole("checkbox"));
    expect(stored(c)).toEqual(Control.stateConfigZ.parse({ hidden: true }));
  });

  it("should keep the rest of the control state when turned off", () => {
    const c = renderField(Control.stateConfigZ.parse({ authority: 100 }));
    fireEvent.click(c.getByRole("checkbox"));
    expect(stored(c)).toEqual(
      Control.stateConfigZ.parse({ authority: 100, hidden: true }),
    );
  });
});
