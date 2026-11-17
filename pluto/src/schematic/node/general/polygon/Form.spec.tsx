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

import { PolygonForm } from "@/schematic/node/general/polygon/Form";
import { renderForm } from "@/testutil";

const path = deep.path<schematic.PolygonNodeConfig>();

describe("PolygonForm", () => {
  it.each([
    { label: "Sides", field: path.numSides, typed: "7.6", expected: 8 },
    { label: "Side length", field: path.sideLength, typed: "24.7", expected: 25 },
    { label: "Border radius", field: path.borderRadius, typed: "4.4", expected: 4 },
  ])("should store a whole $label", async ({ label, field, typed, expected }) => {
    const { getByLabelText, value } = renderForm(
      <PolygonForm />,
      schematic.polygonNodeConfigZ.parse({ variant: "polygon" }),
      schematic.polygonNodeConfigZ,
    );
    const input = getByLabelText(new RegExp(`^${label}`));
    if (!(input instanceof HTMLInputElement)) throw new Error("expected an input");
    fireEvent.change(input, { target: { value: typed } });
    fireEvent.blur(input);
    await waitFor(() => expect(value(field)).toBe(expected));
  });
});
