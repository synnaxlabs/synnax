// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { renderForm } from "@/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Value } from "@/vis/value";

const schema = z.object({ precision: z.number().optional() }).loose();
const path = deep.path<z.infer<typeof schema>>();

const typePrecision = (value: string): void => {
  const input = screen
    .getByText("Precision")
    .closest(".lyra-input__item")
    ?.querySelector("input");
  if (input == null) throw new Error("expected a precision input");
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const renderTelemForm = () =>
  renderForm(
    <Value.TelemForm />,
    { precision: 2 },
    schema,
    createSynnaxWrapper({ client: null }),
  );

describe("Value.TelemForm", () => {
  it("should round a typed fractional precision to a whole number", () => {
    const { value } = renderTelemForm();
    typePrecision("4.6");
    expect(value(path.precision)).toBe(5);
  });

  it("should clamp a typed precision to 17", () => {
    const { value } = renderTelemForm();
    typePrecision("40");
    expect(value(path.precision)).toBe(17);
  });
});
