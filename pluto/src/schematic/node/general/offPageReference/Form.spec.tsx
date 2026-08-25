// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Form } from "@/form";
import { Node } from "@/schematic/node";
import { OffPageReferenceForm } from "@/schematic/node/general/offPageReference/Form";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const CONFIG_Z = schematic.offPageReferenceNodeConfigZ;

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof CONFIG_Z>({
    values: Node.createConfig("off_page_reference"),
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
});
