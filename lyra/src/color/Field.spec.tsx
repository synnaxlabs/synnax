// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Form } from "@/form";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";

const schema = z.object({ color: color.colorZ.optional() });

const RED = "#ff0000";
const BLUE = "#0000ff";

interface HarnessProps {
  values: z.infer<typeof schema>;
  onChange: (values: z.infer<typeof schema>) => void;
}

const Harness = ({ values, onChange }: HarnessProps): ReactElement => {
  const methods = Form.use<typeof schema>({
    values,
    schema,
    onChange: ({ values }) => onChange(values),
  });
  return (
    <Triggers.Provider>
      <Theming.Provider>
        <Form.Form<typeof schema> {...methods}>
          <Color.Field path="color" fallback={RED} />
        </Form.Form>
      </Theming.Provider>
    </Triggers.Provider>
  );
};

const swatchOf = (c: RenderResult): HTMLElement => {
  const el = c.container.querySelector<HTMLElement>(`.${CSS.B("color-swatch")}`);
  if (el == null) throw new Error("no swatch rendered");
  return el;
};

describe("Field", () => {
  it("should show the fallback marked as auto while the field is absent", () => {
    const c = render(<Harness values={{}} onChange={vi.fn()} />);
    expect(swatchOf(c).className).toContain(CSS.M("auto"));
  });

  it("should show a set color without the auto mark", () => {
    const c = render(
      <Harness values={{ color: color.construct(BLUE) }} onChange={vi.fn()} />,
    );
    expect(swatchOf(c).className).not.toContain(CSS.M("auto"));
  });

  it("should write a picked color to the form", () => {
    const onChange = vi.fn();
    const c = render(<Harness values={{}} onChange={onChange} />);
    fireEvent.click(swatchOf(c));
    fireEvent.change(c.getByLabelText("Hex"), { target: { value: "0000ff" } });
    expect(color.hex(onChange.mock.calls.at(-1)?.[0].color)).toEqual(BLUE);
  });

  it("should clear the field when the user picks Auto", () => {
    const onChange = vi.fn();
    const c = render(
      <Harness values={{ color: color.construct(BLUE) }} onChange={onChange} />,
    );
    fireEvent.click(swatchOf(c));
    fireEvent.click(c.getByLabelText("Auto"));
    expect(onChange.mock.calls.at(-1)?.[0].color).toBeUndefined();
    expect(swatchOf(c).className).toContain(CSS.M("auto"));
  });
});
