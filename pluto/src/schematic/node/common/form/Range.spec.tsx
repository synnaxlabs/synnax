// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { type bounds, deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useImperativeHandle } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";

const schema = z.object({
  bounds: z.object({ lower: z.number(), upper: z.number() }),
});

const path = deep.path<z.infer<typeof schema>>();

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
    get: () => methods.get(path.bounds).value,
  }));
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.RangeField path={path.bounds} />
    </Base.Form>
  );
};

interface Fields {
  lower: HTMLInputElement;
  upper: HTMLInputElement;
  get: () => bounds.Bounds;
  invalid: () => boolean;
}

const setup = (initial: bounds.Bounds): Fields => {
  const ref: { current: Handle | null } = { current: null };
  const c = render(<Host initial={initial} ref={ref} />);
  const field = (label: string): HTMLInputElement => {
    const el = c.getByLabelText(label);
    assert(el instanceof HTMLInputElement);
    return el;
  };
  return {
    lower: field("Range start"),
    upper: field("Range end"),
    get: () => {
      assert(ref.current != null);
      return ref.current.get();
    },
    invalid: () => c.container.querySelector(".lyra--error") != null,
  };
};

const commit = (input: HTMLInputElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

describe("Form.RangeField", () => {
  it("should display the current lower and upper values", () => {
    const { lower, upper } = setup({ lower: 0, upper: 100 });
    expect(lower.value).toBe("0");
    expect(upper.value).toBe("100");
  });

  it("should hold a start at or above the end with an error", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "100");
    expect(f.get()).toEqual({ lower: 0, upper: 100 });
    expect(f.lower.value).toBe("100");
    expect(f.invalid()).toBe(true);
  });

  it("should hold an end at or below the start with an error", () => {
    const f = setup({ lower: 20, upper: 100 });
    commit(f.upper, "-50");
    expect(f.get()).toEqual({ lower: 20, upper: 100 });
    expect(f.upper.value).toBe("-50");
    expect(f.invalid()).toBe(true);
  });

  it("should clear the error after a valid edit", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "150");
    commit(f.lower, "40");
    expect(f.get().lower).toBe(40);
    expect(f.invalid()).toBe(false);
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
    expect(f.get().lower).toBe(0);
  });

  it("should allow negative bounds", () => {
    const f = setup({ lower: 0, upper: 100 });
    commit(f.lower, "-200");
    expect(f.get().lower).toBe(-200);
  });
});
