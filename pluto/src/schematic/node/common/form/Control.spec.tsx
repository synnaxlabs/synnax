// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Control } from "@/schematic/node/common/control";
import { ControlChipField } from "@/schematic/node/common/form/Control";

const schema = z.object({ control: Control.stateConfigZ.optional() });

const Stored = (): ReactElement => {
  const control = Form.useFieldValue<Control.StateConfig | undefined>("control", {
    optional: true,
  });
  return <div data-testid="stored">{JSON.stringify(control ?? null)}</div>;
};

const renderField = (control?: Control.StateConfig) => {
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

const stored = (c: ReturnType<typeof render>): Control.StateConfig | null =>
  JSON.parse(c.getByTestId("stored").textContent ?? "null");

describe("ControlChipField", () => {
  it("should read as on for a symbol with no control state", () => {
    const c = renderField(undefined);
    expect(c.getByRole("checkbox")).toHaveProperty("checked", true);
    expect(stored(c)).toBeNull();
  });

  it("should hide the control state of a symbol with none when turned off", () => {
    const c = renderField(undefined);
    fireEvent.click(c.getByRole("checkbox"));
    expect(stored(c)?.hidden).toBe(true);
  });

  it("should keep the rest of the control state when turned off", () => {
    const c = renderField(Control.stateConfigZ.parse({ authority: 100 }));
    fireEvent.click(c.getByRole("checkbox"));
    expect(stored(c)).toEqual(
      Control.stateConfigZ.parse({ authority: 100, hidden: true }),
    );
  });
});
