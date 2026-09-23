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
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Form } from "@/form";
import { Node } from "@/schematic/node";
import { OffPageReferenceForm } from "@/schematic/node/general/offPageReference/Form";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const CONFIG_Z = schematic.offPageReferenceNodeConfigZ;

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof CONFIG_Z>({
    values: Node.createConfig({ variant: "off_page_reference" }),
    schema: CONFIG_Z,
  });
  return (
    <SynnaxWrapper>
      <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
    </SynnaxWrapper>
  );
};

describe("OffPageReferenceForm", () => {
  it("should render the form with label, page, and click mode fields", () => {
    const { getAllByText, getByText } = render(
      <FormWrapper>
        <OffPageReferenceForm />
      </FormWrapper>,
    );
    expect(getAllByText("Label").length).toBeGreaterThan(0);
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
    expect(getByText("Size")).toBeDefined();
  });

  interface PageFormFixtureArgs {
    targetType?: "schematic" | "lineplot";
    initialPage?: (targetKey: string) => schematic.Page | undefined;
  }

  const createPageFormFixture = async ({
    targetType = "lineplot",
    initialPage = () => undefined,
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
    const initialColor = "#000000";
    let methods: Form.UseReturn<typeof CONFIG_Z> | undefined;
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => {
      methods = Form.use<typeof CONFIG_Z>({
        values: CONFIG_Z.parse({
          variant: "off_page_reference",
          label: { label: "Test Label" },
          color: initialColor,
          page: initialPage(target.key),
        }),
        schema: CONFIG_Z,
      });
      return (
        <SynnaxWrapper>
          <Form.Form<typeof CONFIG_Z> {...methods}>{children}</Form.Form>
        </SynnaxWrapper>
      );
    };
    const rendered = render(
      <Wrapper>
        <OffPageReferenceForm schematicKey={source.key} />
      </Wrapper>,
    );
    const getMethods = (): Form.UseReturn<typeof CONFIG_Z> => {
      if (methods == null) throw new Error("form did not mount");
      return methods;
    };
    const getColor = (): string => JSON.stringify(getMethods().get("color").value);
    return { ...rendered, target, targetName, getMethods, getColor };
  };

  it("should select the entry for a schematic page", async () => {
    const { getByText, targetName } = await createPageFormFixture({
      targetType: "schematic",
      initialPage: (key) => ({ type: "schematic", key }),
    });
    await waitFor(() => expect(getByText(targetName)).toBeDefined());
  });

  it("should select the entry for a line plot page", async () => {
    const { getByText, targetName } = await createPageFormFixture({
      initialPage: (key) => ({ type: "lineplot", key }),
    });
    await waitFor(() => expect(getByText(targetName)).toBeDefined());
  });

  it("should write a typed page and recolor when a page is first selected", async () => {
    const { getByText, findByText, getMethods, getColor, target, targetName } =
      await createPageFormFixture();
    const before = getColor();
    fireEvent.click(getByText("Select page"));
    fireEvent.click(await findByText(targetName));
    await waitFor(() =>
      expect(getMethods().get("page").value).toEqual({
        type: "lineplot",
        key: target.key,
      }),
    );
    expect(getColor()).not.toBe(before);
  });

  it("should not recolor when replacing an existing page", async () => {
    const { getByText, findByText, getMethods, getColor, target, targetName } =
      await createPageFormFixture({
        initialPage: () => ({ type: "schematic", key: uuid.create() }),
      });
    const before = getColor();
    fireEvent.click(getByText("Select page"));
    fireEvent.click(await findByText(targetName));
    await waitFor(() =>
      expect(getMethods().get("page").value).toEqual({
        type: "lineplot",
        key: target.key,
      }),
    );
    expect(getColor()).toBe(before);
  });

  it("should clear the page and keep the color when deselected", async () => {
    const { findByText, getAllByText, getMethods, getColor, targetName } =
      await createPageFormFixture({
        initialPage: (key) => ({ type: "lineplot", key }),
      });
    const before = getColor();
    fireEvent.click(await findByText(targetName));
    const options = getAllByText(targetName);
    fireEvent.click(options[options.length - 1]);
    await waitFor(() =>
      expect(getMethods().get("page", { optional: true })?.value).toBeUndefined(),
    );
    expect(getColor()).toBe(before);
  });
});
