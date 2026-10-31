// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { type border } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useImperativeHandle } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";

const xyZ = z.object({ x: z.number(), y: z.number() });
const schema = z.object({
  borderRadius: z
    .object({ topLeft: xyZ, topRight: xyZ, bottomLeft: xyZ, bottomRight: xyZ })
    .optional(),
});

const corners = (x: number, y: number): border.Radius => ({
  topLeft: { x, y },
  topRight: { x, y },
  bottomLeft: { x, y },
  bottomRight: { x, y },
});

const FALLBACK = corners(50, 10);

interface Handle {
  get: () => border.Radius | undefined;
}

interface HostProps {
  initial?: border.Radius;
  ref?: React.Ref<Handle>;
}

const Host = ({ initial, ref }: HostProps): ReactElement => {
  const methods = Base.use({ values: { borderRadius: initial }, schema });
  useImperativeHandle(ref, () => ({
    get: () => methods.get<border.Radius | undefined>("borderRadius").value,
  }));
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.RadiusFields path="borderRadius" fallback={FALLBACK} />
    </Base.Form>
  );
};

const setup = (initial?: border.Radius) => {
  const ref: { current: Handle | null } = { current: null };
  const c = render(<Host initial={initial} ref={ref} />);
  const field = (label: string): HTMLInputElement => {
    const el = c.getByLabelText(new RegExp(label));
    assert(el instanceof HTMLInputElement);
    return el;
  };
  return {
    x: field("X border radius"),
    y: field("Y border radius"),
    get: () => {
      assert(ref.current != null);
      return ref.current.get();
    },
  };
};

describe("Form.RadiusFields", () => {
  it("should show the fallback while the radius is absent", () => {
    const { x, y, get } = setup();
    expect(x.value).toBe("50");
    expect(y.value).toBe("10");
    expect(get()).toBeUndefined();
  });

  it("should show a stored 0", () => {
    const { x, y } = setup(corners(0, 0));
    expect(x.value).toBe("0");
    expect(y.value).toBe("0");
  });

  it("should keep the fallback of the other axis when one is edited", () => {
    const f = setup();
    fireEvent.change(f.x, { target: { value: "0" } });
    fireEvent.blur(f.x);
    expect(f.get()).toEqual(corners(0, 10));
  });
});
