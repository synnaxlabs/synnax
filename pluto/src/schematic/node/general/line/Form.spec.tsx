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

import { LineForm } from "@/schematic/node/general/line/Form";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.LineNodeConfig>();

const input = (container: HTMLElement): HTMLInputElement => {
  const field = container.querySelector<HTMLInputElement>("input[type='text']");
  if (field == null) throw new Error("expected a numeric input");
  return field;
};

describe("LineForm", () => {
  it.each([
    { name: "below the minimum", typed: "0", expected: 1 },
    { name: "above the maximum", typed: "21", expected: 20 },
  ])("should clamp a stroke width $name", async ({ typed, expected }) => {
    const { container, value } = renderForm(
      <LineForm />,
      schematic.lineNodeConfigZ.parse({ variant: "line", strokeWidth: 4 }),
      schematic.lineNodeConfigZ,
    );
    const field = input(container);
    fireEvent.change(field, { target: { value: typed } });
    fireEvent.blur(field);
    await waitFor(() => expect(value(path.strokeWidth)).toBe(expected));
  });
});
