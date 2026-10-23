// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Orientation } from "@/schematic/node/common/orientation";

const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use({
    values: { node: { orientation: "top", label: { orientation: "top" } } },
  });
  return <Form.Form {...methods}>{children}</Form.Form>;
};

describe("Orientation.Section", () => {
  it("should render the field under a layout title", () => {
    render(<Orientation.Section path="node" />, { wrapper: FormWrapper });
    expect(screen.getByText("Layout")).toBeTruthy();
  });

  it("should render nothing when both the inner and outer parts hide", () => {
    const { container } = render(
      <Orientation.Section path="node" hideInner hideOuter />,
      { wrapper: FormWrapper },
    );
    expect(container.innerHTML).toBe("");
  });
});
