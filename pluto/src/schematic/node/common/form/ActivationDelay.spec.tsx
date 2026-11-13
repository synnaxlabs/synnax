// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/schematic/node/common/form";

const schema = z.object({ onClickDelay: z.number().default(0) });

let methods: Base.ContextValue<typeof schema>;

const Host = ({ values }: { values: z.infer<typeof schema> }): ReactElement => {
  methods = Base.use({ values, schema });
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.ActivationDelayField />
    </Base.Form>
  );
};

describe("ActivationDelayField", () => {
  it("should show the saved delay", () => {
    const c = render(
      <Host values={{ onClickDelay: TimeSpan.milliseconds(250).nanoseconds }} />,
    );
    expect(c.getByLabelText(/Delay/)).toHaveProperty("value", "250ms");
  });

  it("should show the schema default when the config has no key", () => {
    const c = render(<Host values={{} as z.infer<typeof schema>} />);
    expect(c.getByLabelText(/Delay/)).toHaveProperty("value", "0s");
  });

  const typeDelay = (value: string): void => {
    const input = screen.getByLabelText<HTMLInputElement>(/Delay/);
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);
  };

  it.each(["250ms", "250"])("should store %s as nanoseconds", (typed) => {
    render(<Host values={{ onClickDelay: 0 }} />);
    typeDelay(typed);
    expect(methods.value().onClickDelay).toBe(250_000_000);
  });

  it("should accept the longest delay that the Core stores", () => {
    render(<Host values={{ onClickDelay: 0 }} />);
    typeDelay("2147483647");
    expect(methods.value().onClickDelay).toBe(2_147_483_647_000_000);
  });

  it("should reject a delay longer than the Core stores", () => {
    render(<Host values={{ onClickDelay: 0 }} />);
    const input = screen.getByLabelText<HTMLInputElement>(/Delay/);
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "2147483648" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(screen.getByText(/^Delay must be .* or less$/)).toBeDefined();
    expect(methods.value().onClickDelay).toBe(0);
  });
});
