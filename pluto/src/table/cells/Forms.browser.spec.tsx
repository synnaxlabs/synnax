// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { deep } from "@synnaxlabs/x";
import type { PropsWithChildren, ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { ValueForm } from "@/table/cells/Forms";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const path = deep.path<table.ValueCellConfig>();

const SynnaxWrapper = createSynnaxWrapper({ client: null });

let methods: Form.ContextValue<typeof table.valueCellConfigZ>;

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  methods = Form.use<typeof table.valueCellConfigZ>({
    values: table.valueCellConfigZ.parse({ variant: "value" }),
    schema: table.valueCellConfigZ,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof table.valueCellConfigZ> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

describe("ValueForm telemetry tab", () => {
  it("should write the staleness timeout to the cell config in seconds", async () => {
    await renderThemed(
      <FormWrapper>
        <ValueForm />
      </FormWrapper>,
    );
    await page.getByText("Telemetry", { exact: true }).click();
    const input = page.getByLabelText("Timeout");
    await userEvent.fill(input, "7s");
    input.element().blur();
    await expect.poll(() => methods.get(path.stalenessTimeout).value).toBe(7);
  });
});
