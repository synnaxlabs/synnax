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
import { act, fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { TextForm, ValueForm } from "@/table/cells/Forms";
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
      <ValueForm />
    </FormWrapper>,
  );
  fireEvent.click(result.getByText(tab));
  return result;
};

const renderTelemetryTab = () => renderTab("Telemetry");

describe("ValueForm", () => {
  it("should open on the telemetry tab", () => {
    const { getByRole } = render(
      <FormWrapper>
        <ValueForm />
      </FormWrapper>,
    );
    expect(getByRole("tab", { name: "Telemetry" }).ariaSelected).toBe("true");
  });

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
    const bars = (container: HTMLElement) =>
      Array.from(container.querySelectorAll<HTMLElement>(".pluto-redline-form__bar"));

    it("should add a band above the highest threshold", () => {
      const { getByText } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByText("Add band"));
      expect(methods.value().redline.bands.map(({ threshold }) => threshold)).toEqual([
        0, 1,
      ]);
    });

    it("should remove a band", () => {
      const { getByText, getByLabelText } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByLabelText("Remove band"));
      expect(methods.value().redline.bands).toEqual([]);
    });

    it("should toggle whether a band flashes", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      const toggle = container.querySelector(
        ".pluto-redline-form__bands [aria-pressed]",
      );
      expect(toggle).not.toBeNull();
      fireEvent.click(toggle!);
      expect(methods.value().redline.bands[0].flashing).toBe(true);
    });

    it("should flash the bar of a flashing band", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      const bar = bars(container)[1];
      expect(bar.classList).not.toContain("pluto--flashing");
      fireEvent.click(
        container.querySelector(".pluto-redline-form__bands [aria-pressed]")!,
      );
      expect(bar.classList).toContain("pluto--flashing");
      const on = bar.style.getPropertyValue("--pluto-redline-on");
      const off = bar.style.getPropertyValue("--pluto-redline-off");
      expect(off).not.toEqual(on);
    });

    it("should blend a band's bar into the band above in smooth mode", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByText("Add band"));
      expect(
        bars(container)[1].style.getPropertyValue("--pluto-redline-on"),
      ).not.toContain("gradient");
      fireEvent.click(getByText("Smooth"));
      expect(methods.value().redline.smooth).toBe(true);
      expect(bars(container)[1].style.getPropertyValue("--pluto-redline-on")).toContain(
        "linear-gradient",
      );
    });

    it("should write a committed threshold", () => {
      const { getByText, container } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      const input = container.querySelector<HTMLInputElement>(
        ".pluto-redline-form__bands .pluto-list__item input",
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
            ".pluto-redline-form__bands .pluto-list__item input",
          ),
        );
      const commit = (input: HTMLInputElement, value: string) => {
        fireEvent.change(input, { target: { value } });
        fireEvent.blur(input);
      };
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByText("Add band"));
      expect(inputs().map((i) => i.value)).toEqual(["0", "1"]);
      commit(inputs()[0], "5");
      expect(inputs().map((i) => i.value)).toEqual(["1", "5"]);
    });

    it("should flag every band that shares a threshold", () => {
      const { getByText, container } = renderTab("Redline");
      const inputs = () =>
        Array.from(
          container.querySelectorAll<HTMLInputElement>(
            ".pluto-redline-form__bands .pluto-list__item input",
          ),
        );
      const flagged = () =>
        inputs().map((i) => i.parentElement?.classList.contains("pluto--error"));
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByText("Add band"));
      fireEvent.click(getByText("Add band"));
      expect(flagged()).toEqual([false, false, false]);
      fireEvent.change(inputs()[2], { target: { value: "0" } });
      fireEvent.blur(inputs()[2]);
      expect(flagged()).toEqual([true, true, false]);
    });

    it("should show that values below the lowest band paint no fill", () => {
      const { getByText, container } = renderTab("Redline");
      expect(getByText("All values")).toBeDefined();
      fireEvent.click(getByText("Add band"));
      const floor = () =>
        container.querySelector(".pluto-redline-form__floor-value")?.textContent;
      expect(floor()).toBe("0");
      expect(
        container.querySelector(".pluto-redline-form__floor-value")?.classList,
      ).not.toContain("pluto--square");
      expect(container.querySelector(".pluto-redline-form__floor input")).toBeNull();
      expect(getByText("No fill")).toBeDefined();
      act(() => methods.set("units", "psi"));
      expect(floor()).toBe("0 psi");
    });

    it("should show the fill below the lowest band once one is set", () => {
      const { getByText } = renderTab("Redline");
      fireEvent.click(getByText("Add band"));
      act(() => methods.set("fillColor", [0, 255, 0, 1]));
      expect(getByText("Fill")).toBeDefined();
    });
  });
});

describe("TextForm", () => {
  const TextWrapper = ({ children }: PropsWithChildren): ReactElement => {
    const textMethods = Form.use<typeof table.textCellConfigZ>({
      values: table.textCellConfigZ.parse({ variant: "text", fillColor: "#00ff00" }),
      schema: table.textCellConfigZ,
    });
    return (
      <SynnaxWrapper>
        <Form.Form<typeof table.textCellConfigZ> {...textMethods}>{children}</Form.Form>
      </SynnaxWrapper>
    );
  };

  it("should edit the text color as auto and show the stored fill", () => {
    const { getAllByText, container } = render(
      <TextWrapper>
        <TextForm />
      </TextWrapper>,
    );
    expect(getAllByText("Text")).toHaveLength(3);
    expect(getAllByText("Fill")).toHaveLength(1);
    const swatches = container.querySelectorAll<HTMLElement>(".pluto-color-swatch");
    expect(swatches).toHaveLength(2);
    expect(swatches[0].className).toContain("pluto--auto");
    expect(swatches[1].className).not.toContain("pluto--auto");
  });
});
