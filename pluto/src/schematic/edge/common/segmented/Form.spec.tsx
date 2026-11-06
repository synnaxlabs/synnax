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
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createDefaultConfig } from "@/schematic/edge/common/segmented/config";
import { Form } from "@/schematic/edge/common/segmented/Form";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.PipeEdgeConfig>();

const renderPipeForm = (config: schematic.PipeEdgeConfig) =>
  renderForm(<Form />, config, schematic.pipeEdgeConfigZ);

const input = (container: HTMLElement): HTMLInputElement => {
  const field = container.querySelector<HTMLInputElement>("input[type='text']");
  if (field == null) throw new Error("expected a numeric input");
  return field;
};

describe("Segmented form", () => {
  it("should show the default stroke width for an edge with none", () => {
    const { container, value } = renderPipeForm(createDefaultConfig("pipe"));
    expect(input(container).value).toBe("2");
    expect(value(path.strokeWidth)).toBeUndefined();
  });

  it("should show the stroke width of an edge that has one", () => {
    const { container } = renderPipeForm({
      ...createDefaultConfig("pipe"),
      strokeWidth: 6,
    });
    expect(input(container).value).toBe("6");
  });

  it("should write a changed stroke width to the config", async () => {
    const { container, value } = renderPipeForm(createDefaultConfig("pipe"));
    const field = input(container);
    fireEvent.change(field, { target: { value: "6" } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.strokeWidth)).toBe(6));
  });
});
