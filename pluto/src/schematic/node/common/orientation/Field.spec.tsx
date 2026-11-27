// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Orientation } from "@/schematic/node/common/orientation";
import { renderForm } from "@/testutil";

const path = deep.path<{ node: schematic.LabeledConfig }>();

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use({
    values: { node: { orientation: "top", label: { orientation: "top" } } },
  });
  return <Form.Form {...methods}>{children}</Form.Form>;
};

describe("Orientation.Field", () => {
  it("should render the field with its label", () => {
    render(<Orientation.Field path={path.node} label="Position" />, {
      wrapper: FormWrapper,
    });
    expect(screen.getByText("Position")).toBeTruthy();
  });

  it("should render nothing when both the inner and outer parts hide", () => {
    const { container } = render(
      <Orientation.Field path={path.node} hideInner hideOuter />,
      { wrapper: FormWrapper },
    );
    expect(container.innerHTML).toBe("");
  });

  it("should store the side that an arrow picks and keep the label", () => {
    const { getByRole, value } = renderForm(
      <Orientation.Field path={path.node} hideOuter />,
      { node: { orientation: "top", label: { orientation: "bottom" } } },
      z.object({ node: z.object({}).loose() }),
    );
    fireEvent.click(getByRole("button", { name: "Right" }));
    expect(value(path.node)).toEqual({
      orientation: "right",
      label: { orientation: "bottom" },
    });
  });
});
