// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Form } from "@/schematic/node/common/form";
import { createSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.CircleNodeConfig>();

const renderField = (config: schematic.CircleNodeConfig) =>
  renderForm(
    <Form.StrokeWidthField />,
    config,
    schematic.circleNodeConfigZ,
    createSynnaxWrapper({ client: null }),
  );

describe("Form.StrokeWidthField", () => {
  it("should show 2 while the stroke width is absent", () => {
    const { getByRole, value } = renderField(
      schematic.circleNodeConfigZ.parse({ variant: "circle" }),
    );
    expect(getByRole("textbox")).toHaveProperty("value", "2");
    expect(value(path.strokeWidth)).toBeUndefined();
  });

  it("should store a stroke width of 0", () => {
    const { getByRole, value } = renderField(
      schematic.circleNodeConfigZ.parse({ variant: "circle", strokeWidth: 3 }),
    );
    const input = getByRole("textbox");
    fireEvent.change(input, { target: { value: "0" } });
    fireEvent.blur(input);
    expect(value(path.strokeWidth)).toBe(0);
  });
});
