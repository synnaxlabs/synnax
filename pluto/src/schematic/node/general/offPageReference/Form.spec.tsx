// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep, uuid } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { OffPageReferenceForm } from "@/schematic/node/general/offPageReference/Form";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const offPageRefSchema = z.object({
  label: z.object({
    label: z.string(),
    level: z.string().optional(),
    orientation: z.string().optional(),
  }),
  page: z.object({ type: z.string(), key: z.string() }).or(z.string()),
  dblClickNav: z.boolean(),
  color: z.string().nullable().optional(),
  orientation: z.string().optional(),
});

const offPageRefValues: z.infer<typeof offPageRefSchema> = {
  label: { label: "Test Label", level: "p", orientation: "top" },
  page: "",
  dblClickNav: true,
  color: "#000000",
  orientation: "left",
};

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof offPageRefSchema>({
    values: deep.copy(offPageRefValues),
    schema: offPageRefSchema,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof offPageRefSchema> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

describe("OffPageReferenceForm", () => {
  it("should render the form with label, page, and click mode fields", () => {
    const { getByText } = render(
      <FormWrapper>
        <OffPageReferenceForm />
      </FormWrapper>,
    );
    expect(getByText("Label")).toBeDefined();
    expect(getByText("Page")).toBeDefined();
    expect(getByText("Click mode")).toBeDefined();
  });

  it("should render single and double click mode buttons", () => {
    const { getByText } = render(
      <FormWrapper>
        <OffPageReferenceForm />
      </FormWrapper>,
    );
    expect(getByText("Single")).toBeDefined();
    expect(getByText("Double")).toBeDefined();
  });

  it("should render color control", () => {
    const { getByText } = render(
      <FormWrapper>
        <OffPageReferenceForm />
      </FormWrapper>,
    );
    expect(getByText("Color")).toBeDefined();
  });

  it("should render label size field when level is provided", () => {
    const { getByText } = render(
      <FormWrapper>
        <OffPageReferenceForm />
      </FormWrapper>,
    );
    expect(getByText("Label size")).toBeDefined();
  });

  interface PageFormFixtureArgs {
    targetType?: "schematic" | "lineplot";
    initialPage?: (targetKey: string) => z.input<typeof offPageRefSchema>["page"];
  }

  const createPageFormFixture = async ({
    targetType = "lineplot",
    initialPage = () => "",
  }: PageFormFixtureArgs = {}) => {
    const client = createTestClient();
    const SynnaxWrapper = await createAsyncSynnaxWrapper({ client });
    const proj = await client.projects.create({ name: "off_page_form", layout: {} });
    const source = await client.schematics.create(proj.key, { name: "source" });
    const targetName = `target_${uuid.create().slice(0, 8)}`;
    const target =
      targetType === "schematic"
        ? await client.schematics.create(proj.key, { name: targetName })
        : await client.lineplots.create(proj.key, { name: targetName });
    let methods: Form.UseReturn<typeof offPageRefSchema> | undefined;
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => {
      methods = Form.use<typeof offPageRefSchema>({
        values: { ...deep.copy(offPageRefValues), page: initialPage(target.key) },
        schema: offPageRefSchema,
      });
      return (
        <SynnaxWrapper>
          <Form.Form<typeof offPageRefSchema> {...methods}>{children}</Form.Form>
        </SynnaxWrapper>
      );
    };
    const rendered = render(
      <Wrapper>
        <OffPageReferenceForm schematicKey={source.key} />
      </Wrapper>,
    );
    const getMethods = (): Form.UseReturn<typeof offPageRefSchema> => {
      if (methods == null) throw new Error("form did not mount");
      return methods;
    };
    return { ...rendered, target, targetName, getMethods };
  };

  it("should select the canonical entry for a legacy bare page value", async () => {
    const { getByText, targetName } = await createPageFormFixture({
      targetType: "schematic",
      initialPage: (key) => key,
    });
    await waitFor(() => expect(getByText(targetName)).toBeDefined());
  });

  it("should select the entry for a page object value", async () => {
    const { getByText, targetName } = await createPageFormFixture({
      initialPage: (key) => ({ type: "lineplot", key }),
    });
    await waitFor(() => expect(getByText(targetName)).toBeDefined());
  });

  it("should write a typed page and recolor when a page is first selected", async () => {
    const { getByText, findByText, getMethods, target, targetName } =
      await createPageFormFixture();
    fireEvent.click(getByText("Select page"));
    fireEvent.click(await findByText(targetName));
    await waitFor(() =>
      expect(getMethods().get("page").value).toEqual({
        type: "lineplot",
        key: target.key,
      }),
    );
    expect(getMethods().get("color").value).not.toBe(offPageRefValues.color);
  });

  it("should not recolor when replacing an existing legacy page", async () => {
    const { getByText, findByText, getMethods, target, targetName } =
      await createPageFormFixture({ initialPage: () => uuid.create() });
    fireEvent.click(getByText("Select page"));
    fireEvent.click(await findByText(targetName));
    await waitFor(() =>
      expect(getMethods().get("page").value).toEqual({
        type: "lineplot",
        key: target.key,
      }),
    );
    expect(getMethods().get("color").value).toBe(offPageRefValues.color);
  });

  it("should clear the page and keep the color when deselected", async () => {
    const { findByText, getAllByText, getMethods, targetName } =
      await createPageFormFixture({
        initialPage: (key) => ({ type: "lineplot", key }),
      });
    fireEvent.click(await findByText(targetName));
    const options = getAllByText(targetName);
    fireEvent.click(options[options.length - 1]);
    await waitFor(() => expect(getMethods().get("page").value).toBe(""));
    expect(getMethods().get("color").value).toBe(offPageRefValues.color);
  });
});
