// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useImperativeHandle } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";

const schema = z.object({ strokeWidth: z.number().optional() });

interface Handle {
  get: () => number | undefined;
}

interface HostProps {
  initial?: number;
  ref?: React.Ref<Handle>;
}

const Host = ({ initial, ref }: HostProps): ReactElement => {
  const methods = Base.use({ values: { strokeWidth: initial }, schema });
  useImperativeHandle(ref, () => ({
    get: () => methods.get<number | undefined>("strokeWidth").value,
  }));
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.NumericField path="strokeWidth" label="Stroke width" fallback={2} />
    </Base.Form>
  );
};

const setup = (initial?: number) => {
  const ref: { current: Handle | null } = { current: null };
  const c = render(<Host initial={initial} ref={ref} />);
  const input = c.getByLabelText(/Stroke width/);
  assert(input instanceof HTMLInputElement);
  return {
    input,
    get: () => {
      assert(ref.current != null);
      return ref.current.get();
    },
  };
};

describe("Form.NumericField", () => {
  it("should show the fallback while the field is absent", () => {
    const { input, get } = setup();
    expect(input.value).toBe("2");
    expect(get()).toBeUndefined();
  });

  it("should show a stored 0", () => {
    expect(setup(0).input.value).toBe("0");
  });

  it("should store a 0 typed into an absent field", () => {
    const { input, get } = setup();
    fireEvent.change(input, { target: { value: "0" } });
    fireEvent.blur(input);
    expect(get()).toBe(0);
  });
});
