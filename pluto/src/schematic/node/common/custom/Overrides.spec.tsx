// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { Custom } from "@/schematic/node/common/custom";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const formSchema = z.object({
  specKey: z.string(),
  stateOverrides: schematic.symbol.stateZ.array(),
});

const createState = (key: string, name: string): schematic.symbol.State => ({
  key,
  name,
  regions: [
    {
      key: "main",
      name: `${name} Region`,
      selectors: [".main"],
      strokeColor: color.construct("#333333"),
      fillColor: color.construct("#cccccc"),
    },
  ],
});

describe("StateOverrideForm", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createSymbol = async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: "state-override-form-spec",
    });
    return await client.schematics.symbols.create({
      name: "actuated",
      parent: group.ontologyID(parent.key),
      data: {
        svg: '<svg viewBox="0 0 10 10"><rect class="main" /></svg>',
        states: [createState("base", "Base"), createState("active", "Active")],
        handles: [],
        variant: "actuator",
        scale: 1,
        strokeScaled: false,
        previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
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

  it("should render the resolved symbol's states and regions", async () => {
    const symbol = await createSymbol();
    const { getByText } = renderForm(symbol.key);
    await waitFor(() => expect(getByText("Base Region")).toBeTruthy());
    expect(getByText("Active")).toBeTruthy();
  });

  it("should fall back to the first state when the selected one disappears", async () => {
    const symbol = await createSymbol();
    const { getByText, queryByText, form } = renderForm(symbol.key);
    await waitFor(() => expect(getByText("Active")).toBeTruthy());
    fireEvent.click(getByText("Active"));
    await waitFor(() => expect(getByText("Active Region")).toBeTruthy());
    // Swapping the overrides beneath the held selection must not crash the
    // region list; it falls back to the first remaining state.
    act(() => form().set("stateOverrides", [createState("base", "Base")]));
    await waitFor(() => expect(getByText("Base Region")).toBeTruthy());
    expect(queryByText("Active Region")).toBeNull();
  });
});
