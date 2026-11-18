// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color, deep } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Pipe } from "@/schematic/edge/pipe";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.PipeEdgeConfig>();

const CONFIG = Pipe.spec.defaultConfig();

const renderPipeForm = (config: schematic.PipeEdgeConfig) =>
  renderForm(<Pipe.spec.Form />, config, Pipe.spec.configZ);

const input = (container: HTMLElement): HTMLInputElement => {
  const field = container.querySelector<HTMLInputElement>("input[type='text']");
  if (field == null) throw new Error("expected a numeric input");
  return field;
};

describe("Segmented form", () => {
  it("should write a typed stroke as a fixed color reference", async () => {
    const { getAllByLabelText, value } = renderPipeForm(CONFIG);
    const hex = getAllByLabelText("Stroke").find(
      (el): el is HTMLInputElement => el instanceof HTMLInputElement,
    );
    if (hex == null) throw new Error("expected a stroke input");
    fireEvent.change(hex, { target: { value: "ff0000" } });
    await waitFor(() =>
      expect(value(path.strokeColor)).toEqual({
        type: "fixed",
        value: color.construct("#ff0000"),
      }),
    );
  });

  it("should show the default stroke width of a new edge", () => {
    const { container } = renderPipeForm(CONFIG);
    expect(input(container).value).toBe("2");
  });

  it("should show the stroke width of an edge that has one", () => {
    const { container } = renderPipeForm({ ...CONFIG, strokeWidth: 6 });
    expect(input(container).value).toBe("6");
  });

  it("should write a changed stroke width to the config", async () => {
    const { container, value } = renderPipeForm(CONFIG);
    const field = input(container);
    fireEvent.change(field, { target: { value: "6" } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.strokeWidth)).toBe(6));
  });

  it.each([
    { name: "above the maximum", typed: "21", expected: 20 },
    { name: "below the minimum", typed: "0", expected: 1 },
    { name: "with a fraction", typed: "2.6", expected: 3 },
  ])("should round and clamp a stroke width $name", async ({ typed, expected }) => {
    const { container, value } = renderPipeForm(CONFIG);
    const field = input(container);
    fireEvent.change(field, { target: { value: typed } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.strokeWidth)).toBe(expected));
  });
});
