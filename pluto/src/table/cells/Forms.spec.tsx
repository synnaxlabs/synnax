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
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

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

const renderTab = (tab: string) => {
  const result = render(
    <FormWrapper>
      <ValueForm onVariantChange={vi.fn()} />
    </FormWrapper>,
  );
  fireEvent.click(result.getByText(tab));
  return result;
};

const renderTelemetryTab = () => renderTab("Telemetry");

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

  describe("redline tab", () => {
    it("should add a band above the highest threshold", () => {
      const { getByText } = renderTab("Redline");
      fireEvent.click(getByText("Add a band"));
      fireEvent.click(getByText("Add band"));
      expect(methods.value().redline.bands.map(({ threshold }) => threshold)).toEqual([
        0, 1,
      ]);
    });

    it("should remove a band", () => {
      const { getByText, getByLabelText } = renderTab("Redline");
      fireEvent.click(getByText("Add a band"));
      fireEvent.click(getByLabelText("Remove band"));
      expect(methods.value().redline.bands).toEqual([]);
    });

    it("should toggle whether a band flashes", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add a band"));
      const toggle = container.querySelector("[aria-pressed]");
      expect(toggle).not.toBeNull();
      fireEvent.click(toggle!);
      expect(methods.value().redline.bands[0].flashing).toBe(true);
    });

    it("should write a committed threshold", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add a band"));
      const input = container.querySelector<HTMLInputElement>(
        ".pluto-redline-form__bands input",
      );
      expect(input).not.toBeNull();
      fireEvent.change(input!, { target: { value: "850" } });
      fireEvent.blur(input!);
      expect(methods.value().redline.bands[0].threshold).toBe(850);
    });

    it("should sort the rows again when a threshold commits", () => {
      const { getByText, container } = renderTab("Redline");
      const inputs = () =>
        Array.from(
          container.querySelectorAll<HTMLInputElement>(
            ".pluto-redline-form__bands input",
          ),
        );
      const commit = (input: HTMLInputElement, value: string) => {
        fireEvent.change(input, { target: { value } });
        fireEvent.blur(input);
      };
      fireEvent.click(getByText("Add a band"));
      fireEvent.click(getByText("Add band"));
      expect(inputs().map((i) => i.value)).toEqual(["1", "0"]);
      commit(inputs()[1], "5");
      expect(inputs().map((i) => i.value)).toEqual(["5", "1"]);
    });

    it("should leave the base absent until one is picked", () => {
      const { getByText } = renderTab("Redline");
      fireEvent.click(getByText("Add a band"));
      expect(getByText("Base: no fill")).toBeDefined();
      expect(methods.value().redline.base).toBeUndefined();
    });
  });
});
