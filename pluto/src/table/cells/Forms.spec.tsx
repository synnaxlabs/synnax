// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Form } from "@/form";
import { ValueForm } from "@/table/cells/Forms";
import { createSynnaxWrapper } from "@/testutil/Synnax";

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

const renderTelemetryTab = () => {
  const result = render(
    <FormWrapper>
      <ValueForm onVariantChange={vi.fn()} />
    </FormWrapper>,
  );
  fireEvent.click(result.getByText("Telemetry"));
  return result;
};

describe("ValueForm", () => {
  describe("telemetry tab", () => {
    it("should render the telemetry fields for a value cell", () => {
      const { getByText } = renderTelemetryTab();
      expect(getByText("Channel")).toBeDefined();
      expect(getByText("Notation")).toBeDefined();
      expect(getByText("Precision")).toBeDefined();
      expect(getByText("Averaging window")).toBeDefined();
    });

    it("should write the averaging window to the cell config", () => {
      const { getByText } = renderTelemetryTab();
      const input = getByText("Averaging window")
        .closest(".pluto-input__item")
        ?.querySelector("input");
      expect(input).not.toBeNull();
      fireEvent.change(input!, { target: { value: "7" } });
      fireEvent.blur(input!);
      expect(methods.get<number>("rollingAverage").value).toBe(7);
    });
  });
});
