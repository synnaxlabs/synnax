// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { deep, TimeSpan } from "@synnaxlabs/x";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";
import { z } from "zod";

import { Staleness } from "@/vis/staleness";

const schema = z.object({ stalenessTimeout: z.number().optional() });
type Values = z.infer<typeof schema>;
const path = deep.path<Values>();

let methods: Form.ContextValue<typeof schema>;

const Host = ({ values, unit }: { values: Values; unit?: TimeSpan }): ReactElement => {
  methods = Form.use({ values, schema });
  return (
    <Form.Form<typeof schema> {...methods}>
      <Staleness.Fields unit={unit} />
    </Form.Form>
  );
};

const value = (p: deep.Target<unknown>): unknown =>
  methods.get(p, { optional: true })?.value;

const field = () => page.getByLabelText("Timeout");

const typeTimeout = async (value: string): Promise<void> => {
  await userEvent.fill(field(), value);
  field().element().blur();
};

describe("Staleness.Fields", () => {
  it("should show and store a timeout in nanoseconds by default", async () => {
    await renderThemed(
      <Host values={{ stalenessTimeout: Number(TimeSpan.seconds(3).valueOf()) }} />,
    );
    await expect.element(field()).toHaveValue("3s");
    await typeTimeout("7s");
    await expect
      .poll(() => value(path.stalenessTimeout))
      .toBe(Number(TimeSpan.seconds(7).valueOf()));
  });

  it("should show and store a timeout in the given unit", async () => {
    await renderThemed(
      <Host values={{ stalenessTimeout: 3 }} unit={TimeSpan.SECOND} />,
    );
    await expect.element(field()).toHaveValue("3s");
    await typeTimeout("7s");
    await expect.poll(() => value(path.stalenessTimeout)).toBe(7);
  });

  it("should accept a timeout up to the longest that the Core stores", async () => {
    await renderThemed(
      <Host values={{ stalenessTimeout: 3 }} unit={TimeSpan.SECOND} />,
    );
    await typeTimeout("104d");
    await expect
      .poll(() => value(path.stalenessTimeout))
      .toBe(TimeSpan.days(104).seconds);
  });

  it("should reject a timeout longer than the Core stores", async () => {
    await renderThemed(
      <Host values={{ stalenessTimeout: 3 }} unit={TimeSpan.SECOND} />,
    );
    await userEvent.fill(field(), "105d");
    await userEvent.keyboard("{Enter}");
    await expect
      .element(page.getByText(/^Timeout must be .* or less$/))
      .toBeInTheDocument();
    expect(value(path.stalenessTimeout)).toBe(3);
  });
});
