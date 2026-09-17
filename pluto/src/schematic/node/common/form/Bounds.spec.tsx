// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useImperativeHandle } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";

const schema = z.object({
  bounds: z.object({ lower: z.number(), upper: z.number() }),
});

interface Handle {
  get: () => bounds.Bounds;
}

interface HostProps {
  initial: bounds.Bounds;
  ref?: React.Ref<Handle>;
}

const Host = ({ initial, ref }: HostProps): ReactElement => {
  const methods = Base.use({ values: { bounds: initial }, schema });
  useImperativeHandle(ref, () => ({
    get: () => methods.get<bounds.Bounds>("bounds").value,
  }));
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.BoundsFields path="bounds" />
    </Base.Form>
  );
};

interface Fields {
  lower: HTMLInputElement;
  upper: HTMLInputElement;
  get: () => bounds.Bounds;
}

const setup = (initial: bounds.Bounds): Fields => {
  const ref: { current: Handle | null } = { current: null };
  const c = render(<Host initial={initial} ref={ref} />);
  // The label carries a required indicator, so the accessible name is not exact.
  const field = (label: string): HTMLInputElement => {
    const el = c.getByLabelText(new RegExp(label));
    assert(el instanceof HTMLInputElement);
    return el;
  };
  return {
    lower: field("Min value"),
    upper: field("Max value"),
    get: () => {
      assert(ref.current != null);
      return ref.current.get();
    },
  };
};

const commit = (input: HTMLInputElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

describe("Form.BoundsFields", () => {
  it("should display the current lower and upper values", () => {
    const { lower, upper } = setup({ lower: 0, upper: 100 });
    expect(lower.value).toBe("0");
    expect(upper.value).toBe("100");
  });

  it("should cap the minimum at the current maximum", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "150");
    expect(f.get().lower).toBe(100);
  });

  it("should cap the maximum at the current minimum", () => {
    const f = setup({ lower: 20, upper: 100 });
    commit(f.upper, "-50");
    expect(f.get().upper).toBe(20);
  });

  it("should accept a minimum below the maximum untouched", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "40");
    expect(f.get().lower).toBe(40);
  });

  it("should follow the maximum after it moves", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.upper, "50");
    expect(f.get().upper).toBe(50);
    commit(f.lower, "80");
    expect(f.get().lower).toBe(50);
  });

  it("should allow negative bounds", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "-200");
    expect(f.get().lower).toBe(-200);
  });
});
