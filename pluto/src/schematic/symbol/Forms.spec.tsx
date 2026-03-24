// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { OffPageReferenceForm } from "@/schematic/symbol/Forms";
import { Theming } from "@/theming";

const offPageRefSchema = z.object({
  label: z.object({
    label: z.string(),
    level: z.string().optional(),
    orientation: z.string().optional(),
  }),
  page: z.string(),
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

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof offPageRefSchema>({
    values: deep.copy(offPageRefValues),
    schema: offPageRefSchema,
  });
  return (
    <Theming.Provider>
      <Form.Form<typeof offPageRefSchema> {...methods}>{children}</Form.Form>
    </Theming.Provider>
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
    expect(getByText("Click Mode")).toBeDefined();
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
    expect(getByText("Label Size")).toBeDefined();
  });
});
