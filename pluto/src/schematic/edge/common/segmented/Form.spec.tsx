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

import { Pipe } from "@/schematic/edge/pipe";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.Edge>();

const EDGE = schematic.edgeZ.parse({
  key: "e1",
  source: { node: "a", param: "out" },
  target: { node: "b", param: "in" },
});

const renderPipeForm = (edge: schematic.Edge) =>
  renderForm(<Pipe.spec.Form />, edge, schematic.edgeZ);

const input = (container: HTMLElement): HTMLInputElement => {
  const field = container.querySelector<HTMLInputElement>("input[type='text']");
  if (field == null) throw new Error("expected a numeric input");
  return field;
};

describe("Segmented form", () => {
  it("should show the default stroke width of a new edge", () => {
    const { container } = renderPipeForm(EDGE);
    expect(input(container).value).toBe("2");
  });

  it("should show the stroke width of an edge that has one", () => {
    const { container } = renderPipeForm({
      ...EDGE,
      strokeWidth: 6,
    });
    expect(input(container).value).toBe("6");
  });

  it("should write a changed stroke width to the config", async () => {
    const { container, value } = renderPipeForm(EDGE);
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
    const { container, value } = renderPipeForm(EDGE);
    const field = input(container);
    fireEvent.change(field, { target: { value: typed } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.strokeWidth)).toBe(expected));
  });
});
