// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { color, deep } from "@synnaxlabs/x";
import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { assert, beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const formSchema = z.object({
  specKey: z.string(),
  stateOverrides: schematic.symbol.stateOverrideZ.array(),
});

const path = deep.path<z.infer<typeof formSchema>>();

const fixed = (hex: string) => ({
  type: "fixed" as const,
  value: color.construct(hex),
});

describe("StateOverrideForm", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createSymbol = async () => {
    const e = await client.enumerations.create({
      name: "Actuated",
      entries: [
        { key: "base", value: 0, name: "Base", color: fixed("#333333") },
        { key: "active", value: 1, name: "Active", color: fixed("#00ff00") },
      ],
    });
    return await client.schematics.symbols.create({
      name: "actuated",
      data: {
        drawing: {
          type: "svg",
          markup:
            '<svg viewBox="0 0 10 10">' +
            '<rect class="main" stroke="#111111" fill="#eeeeee" /></svg>',
        },
        parts: [{ key: "main", name: "Main", selectors: [".main"] }],
        states: [
          { type: "enumeration", key: "active", name: "Active", enumeration: e.key },
        ],
        looks: [
          { key: "stroke", part: "main", stroke: { type: "entry", state: "active" } },
          {
            key: "fill",
            part: "main",
            fill: { type: "color", value: fixed("#cccccc") },
          },
        ],
        variant: "actuator",
      },
    });
  };

  const renderForm = (specKey: string) => {
    let form!: Form.UseReturn<typeof formSchema>;
    const Container = (): ReactElement => {
      form = Form.use<typeof formSchema>({
        values: { specKey, stateOverrides: [] },
        schema: formSchema,
      });
      return (
        <Form.Form<typeof formSchema> {...form}>
          <Custom.StateOverrideForm />
        </Form.Form>
      );
    };
    const utils = render(<Container />, { wrapper });
    return { ...utils, form: () => form };
  };

  it("should give each entry an override with the colors of its looks", async () => {
    const symbol = await createSymbol();
    const { getByText, form } = renderForm(symbol.key);
    await waitFor(() => expect(getByText("Main")).toBeTruthy());
    expect(getByText("Active")).toBeTruthy();
    const regions = form()
      .value()
      .stateOverrides.map(({ key, regions }) => [key, regions]);
    expect(regions).toEqual([
      [
        "base",
        [
          {
            key: "main",
            name: "Main",
            selectors: [".main"],
            strokeColor: color.construct("#333333"),
            fillColor: color.construct("#cccccc"),
          },
        ],
      ],
      [
        "active",
        [
          {
            key: "main",
            name: "Main",
            selectors: [".main"],
            strokeColor: color.construct("#00ff00"),
            fillColor: color.construct("#cccccc"),
          },
        ],
      ],
    ]);
  });

  it("should fall back to the first state when the selected one disappears", async () => {
    const symbol = await createSymbol();
    const { getByText, queryByText, form } = renderForm(symbol.key);
    await waitFor(() => expect(getByText("Active")).toBeTruthy());
    fireEvent.click(getByText("Active"));
    // Swapping the overrides beneath the held selection must not crash the
    // region list; it falls back to the first remaining state.
    const [base] = form().value().stateOverrides;
    act(() => form().set(path.stateOverrides, [base]));
    await waitFor(() => expect(queryByText("Active")).toBeNull());
    expect(getByText("Main")).toBeTruthy();
  });

  describe("region colors", () => {
    const baseRegion = (form: Form.UseReturn<typeof formSchema>) => {
      const base = form.value().stateOverrides.find((s) => s.key === "base");
      return base?.regions.find((r) => r.key === "main");
    };

    const renderRegion = async () => {
      const symbol = await createSymbol();
      const utils = renderForm(symbol.key);
      await waitFor(() => expect(utils.getByText("Main")).toBeTruthy());
      const swatches = Array.from(
        utils.container.querySelectorAll<HTMLElement>(".lyra-color-swatch"),
      );
      return { ...utils, swatches };
    };

    it("should show the stroke and fill as swatches with no hex box", async () => {
      const { swatches, queryAllByRole } = await renderRegion();
      expect(swatches).toHaveLength(2);
      expect(queryAllByRole("textbox")).toHaveLength(0);
    });

    it("should set the stroke from its swatch", async () => {
      const { swatches, getByLabelText, form } = await renderRegion();
      fireEvent.click(swatches[0]);
      fireEvent.change(getByLabelText("Hex"), { target: { value: "ff0000" } });
      await waitFor(() =>
        expect(baseRegion(form())?.strokeColor).toEqual(color.construct("#ff0000")),
      );
    });

    it("should reset a changed region to the symbol's definition", async () => {
      const { container, form } = await renderRegion();
      const stroke = path.stateOverrides.base.regions.main.strokeColor;
      act(() => form().set(stroke, color.construct("#ff0000")));
      const reset = container.querySelector<HTMLElement>(
        ".pluto-schematic-region-reset",
      );
      assert(reset != null);
      fireEvent.click(reset);
      await waitFor(() =>
        expect(baseRegion(form())?.strokeColor).toEqual(color.construct("#333333")),
      );
    });

    it("should clear the fill with Auto and show the SVG's own fill", async () => {
      const { swatches, getByLabelText, form } = await renderRegion();
      fireEvent.click(swatches[1]);
      fireEvent.click(getByLabelText("Auto"));
      await waitFor(() => expect(baseRegion(form())?.fillColor).toBeUndefined());
      expect(baseRegion(form())?.strokeColor).toBeDefined();
      expect(swatches[1].style.getPropertyValue("--pluto-swatch-color")).toBe(
        color.cssString("#eeeeee"),
      );
    });
  });
});
