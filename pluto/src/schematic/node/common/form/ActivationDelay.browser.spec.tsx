// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { TimeSpan } from "@synnaxlabs/x";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";
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

const input = () => page.getByLabelText(/Delay/);

describe("ActivationDelayField", () => {
  it("should show the saved delay", async () => {
    await renderThemed(
      <Host values={{ onClickDelay: TimeSpan.milliseconds(250).nanoseconds }} />,
    );
    await expect.element(input()).toHaveValue("250ms");
  });

  it("should show the schema default when the config has no key", async () => {
    await renderThemed(<Host values={{} as z.infer<typeof schema>} />);
    await expect.element(input()).toHaveValue("0s");
  });

  const typeDelay = async (value: string): Promise<void> => {
    await userEvent.fill(input(), value);
    input().element().blur();
  };

  it.each(["250ms", "250"])("should store %s as nanoseconds", async (typed) => {
    await renderThemed(<Host values={{ onClickDelay: 0 }} />);
    await typeDelay(typed);
    await expect.poll(() => methods.value().onClickDelay).toBe(250_000_000);
  });

  it("should accept the longest delay that the Core stores", async () => {
    await renderThemed(<Host values={{ onClickDelay: 0 }} />);
    await typeDelay("2147483647");
    await expect.poll(() => methods.value().onClickDelay).toBe(2_147_483_647_000_000);
  });

  it("should reject a delay longer than the Core stores", async () => {
    await renderThemed(<Host values={{ onClickDelay: 0 }} />);
    await userEvent.fill(input(), "2147483648");
    await userEvent.keyboard("{Enter}");
    await expect
      .element(page.getByText(/^Delay must be .* or less$/))
      .toBeInTheDocument();
    expect(methods.value().onClickDelay).toBe(0);
  });
});
